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
	frameRate = 24000                      // 24kHz sample rate
	frameSize = 960                        // 20ms frame size
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

	// Check if already connected
	if vc, exists := s.voiceConns[guildID]; exists && vc != nil && vc.Ready {
		if vc.ChannelID == channelID {
			return vc, nil
		}
		// Disconnect from old channel
		vc.Close()
	}

	// Connect to voice channel while listening to the channel
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
	// Generate TTS audio
	req := openai.CreateSpeechRequest{
		Model: openai.SpeechModel(s.ttsModel),
		Input: text,
		Voice: openai.VoiceAlloy, // Default voice
	}
	resp, err := s.client.CreateSpeech(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to generate TTS audio: %w", err)
	}
	defer resp.Close()

	// Decode MP3 to PCM
	audio, err := io.ReadAll(resp)
	if err != nil {
		return fmt.Errorf("failed to read TTS audio: %w", err)
	}

	decoder, err := mp3.NewDecoder(bytes.NewReader(audio))
	if err != nil {
		return fmt.Errorf("failed to create MP3 decoder: %w", err)
	}

	// Convert to PCM
	pcm := make([]int16, 0, frameSize*channels)
	byteBuffer := make([]byte, frameSize*channels*2) // 2 bytes per sample (int16)
	for {
		_, err := decoder.Read(byteBuffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode MP3: %w", err)
		}

		// Convert bytes to int16 (little-endian, assuming PCM is 16-bit)

		for i := 0; i < len(pcm)-frameSize*channels; i += frameSize * channels {
			sample := pcm[i : i+frameSize*channels]
			pcm = append(pcm, sample...)
		}
	}

	log.Printf("📢 Decoded PCM: %d samples (expected multiple of %d for %dms frames)", len(pcm), frameSize*channels, frameSize*1000/frameRate)

	// Initialize Opus encoder
	enc, err := opus.NewEncoder(frameRate, channels, opus.AppAudio)
	if err != nil {
		return fmt.Errorf("failed to create Opus encoder: %w", err)
	}

	vc.Speaking(true)
	defer vc.Speaking(false)

	// Stream audio
	for i := 0; i < len(pcm); i += frameSize * channels {
		end := i + frameSize*channels
		if end > len(pcm) {
			end = len(pcm)
		}
		sample := pcm[i:end]

		// Log sample size
		log.Printf("📢 Encoding frame: %d samples (expected %d)", len(sample), frameSize*channels)

		// Pad sample if too short
		if len(sample) < frameSize*channels {
			log.Printf("⚠️ Padding frame with %d zeros to reach %d samples", frameSize*channels-len(sample), frameSize*channels)
			padding := make([]int16, frameSize*channels-len(sample))
			sample = append(sample, padding...)
		}

		// Encode to Opus
		opusData := make([]byte, maxBytes)
		n, err := enc.Encode(sample, opusData)
		if err != nil {
			log.Printf("⚠️ Error encoding audio: %v (sample size: %d)", err, len(sample))
			return fmt.Errorf("error encoding audio: %w", err)
		}
		opusData = opusData[:n]

		// Send Opus frames
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
