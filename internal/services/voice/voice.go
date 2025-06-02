package voice

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/hajimehoshi/go-mp3"
	"github.com/hraban/opus"
	"github.com/sashabaranov/go-openai"
)

const (
	channels  = 2                          // Stereo audio
	frameRate = 48000                      // Updated to 48kHz for better quality
	frameSize = 1920                       // 40ms frame size at 48kHz
	maxBytes  = (frameSize * 2 * channels) // Max bytes per frame
)

type Service struct {
	client     *openai.Client
	ttsModel   string
	voiceConns map[string]*discordgo.VoiceConnection
	voiceMu    sync.Mutex
}

type Config struct {
	OpenAIAPIKey string
	TTSModel     string
}

func NewService(cfg Config) *Service {
	client := openai.NewClient(cfg.OpenAIAPIKey)
	return &Service{
		client:     client,
		ttsModel:   cfg.TTSModel,
		voiceConns: make(map[string]*discordgo.VoiceConnection),
	}
}

// JoinVoiceChannel joins the specified voice channel and stores the connection
func (s *Service) JoinVoiceChannel(ctx context.Context, session *discordgo.Session, guildID, channelID string) (*discordgo.VoiceConnection, error) {
	s.voiceMu.Lock()
	defer s.voiceMu.Unlock()

	if vc, exists := s.voiceConns[guildID]; exists && vc != nil && vc.Ready {
		if vc.ChannelID == channelID {
			return vc, nil
		}
		vc.Close()
	}

	vc, err := session.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to join voice channel: %w", err)
	}

	s.voiceConns[guildID] = vc
	log.Printf("✅ Joined voice channel %s in guild %s", channelID, guildID)
	return vc, nil
}

// SpeakText generates TTS audio and plays it in the voice channel
func (s *Service) SpeakText(ctx context.Context, vc *discordgo.VoiceConnection, text string) error {
	req := openai.CreateSpeechRequest{
		Model: openai.SpeechModel(s.ttsModel),
		Input: text,
		Voice: openai.VoiceAlloy,
	}
	resp, err := s.client.CreateSpeech(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to generate TTS audio: %w", err)
	}
	defer resp.Close()

	audio, err := io.ReadAll(resp)
	if err != nil {
		return fmt.Errorf("failed to read TTS audio: %w", err)
	}
	log.Printf("📢 Received %d bytes of TTS audio", len(audio))

	decoder, err := mp3.NewDecoder(bytes.NewReader(audio))
	if err != nil {
		return fmt.Errorf("failed to create MP3 decoder: %w", err)
	}

	var pcm []int16
	byteBuffer := make([]byte, maxBytes)
	for {
		n, err := decoder.Read(byteBuffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode MP3: %w", err)
		}
		if n == 0 {
			continue
		}

		for i := 0; i < n-1; i += 2 {
			if i+1 >= n {
				break
			}
			sample := int16(byteBuffer[i]) | int16(byteBuffer[i+1])<<8
			pcm = append(pcm, sample)
		}
	}
	log.Printf("📢 Decoded PCM: %d samples (expected multiple of %d for %dms frames)", len(pcm), frameSize*channels, frameSize*1000/frameRate)

	enc, err := opus.NewEncoder(frameRate, channels, opus.AppAudio)
	if err != nil {
		return fmt.Errorf("failed to create Opus encoder: %w", err)
	}
	enc.SetBitrate(64000) // Set bitrate to 64kbps for better quality
	log.Printf("📢 Using encoder: %d Hz, %d channels, %d kbps", frameRate, channels, 64)

	vc.Speaking(true)
	defer vc.Speaking(false)

	for i := 0; i < len(pcm); i += frameSize * channels {
		end := i + frameSize*channels
		if end > len(pcm) {
			end = len(pcm)
		}
		sample := pcm[i:end]

		if len(sample) < frameSize*channels {
			log.Printf("⚠️ Padding frame with %d zeros", frameSize*channels-len(sample))
			padding := make([]int16, frameSize*channels-len(sample))
			sample = append(sample, padding...)
		} else if len(sample) > frameSize*channels {
			sample = sample[:frameSize*channels]
		}

		opusData := make([]byte, maxBytes)
		n, err := enc.Encode(sample, opusData)
		if err != nil {
			log.Printf("⚠️ Error encoding audio: %v (sample size: %d)", err, len(sample))
			return fmt.Errorf("error encoding audio: %w", err)
		}
		opusData = opusData[:n]

		select {
		case vc.OpusSend <- opusData:
			log.Printf("📢 Sent Opus frame: %d bytes", n)
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// DisconnectVoice disconnects from the voice channel in the guild
func (s *Service) DisconnectVoice(guildID string) {
	s.voiceMu.Lock()
	defer s.voiceMu.Unlock()

	if vc, exists := s.voiceConns[guildID]; exists && vc != nil {
		vc.Close()
		delete(s.voiceConns, guildID)
		log.Printf("✅ Disconnected from voice channel in guild %s", guildID)
	}
}
