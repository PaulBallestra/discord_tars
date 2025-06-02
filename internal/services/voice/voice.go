package voice

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/hajimehoshi/go-mp3"
	"github.com/hraban/opus"
	"github.com/sashabaranov/go-openai"
)

const (
	channels  = 2                          // Stereo audio
	frameRate = 24000                      // Match OpenAI TTS output (24kHz)
	frameSize = 480                        // 20ms frame size at 24kHz (480 samples per 20ms)
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

	vc, err := session.ChannelVoiceJoin(guildID, channelID, false, false) // Enable receiving
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

	// Log sample rate for debugging
	log.Printf("🎙️ MP3 sample rate: %d Hz", decoder.SampleRate())

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
	log.Printf("📢 Decoded PCM: %d samples (expected multiple of %d for %dms frames)",
		len(pcm), frameSize*channels, frameSize*1000/frameRate)

	enc, err := opus.NewEncoder(frameRate, channels, opus.AppVoIP)
	if err != nil {
		return fmt.Errorf("failed to create Opus encoder: %w", err)
	}
	enc.SetBitrate(64000)
	if err := enc.SetInBandFEC(true); err != nil {
		log.Printf("⚠️ Failed to enable FEC: %v", err)
	}
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
			padding := make([]int16, frameSize*channels-len(sample))
			log.Printf("⚠️ Padding frame with %d zeros", len(padding))
			sample = append(sample, padding...)
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

// ListenToVoice captures incoming audio, transcribes it using OpenAI Whisper, and returns the text
func (s *Service) ListenToVoice(ctx context.Context, vc *discordgo.VoiceConnection) (string, error) {
	log.Printf("🎧 Starting to listen to voice channel")

	var pcmBuffer []int16
	decoder, err := opus.NewDecoder(frameRate, channels)
	if err != nil {
		return "", fmt.Errorf("failed to create Opus decoder: %w", err)
	}

	// Collect audio for 5 seconds
	timeout := time.After(5 * time.Second)
	for {
		select {
		case data := <-vc.OpusSend:
			if data == nil {
				continue
			}
			log.Printf("🎧 Received Opus frame: %d bytes", len(data))
			pcm := make([]int16, frameSize*channels)
			n, err := decoder.Decode(data, pcm)
			if err != nil {
				log.Printf("⚠️ Error decoding Opus: %v", err)
				continue
			}
			log.Printf("🎧 Decoded %d PCM samples", n)
			pcmBuffer = append(pcmBuffer, pcm[:n]...)
		case <-timeout:
			log.Printf("🎧 Finished collecting audio, total samples: %d", len(pcmBuffer))
			goto transcription
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

transcription:
	if len(pcmBuffer) == 0 {
		return "", fmt.Errorf("no audio data collected")
	}

	// Convert PCM to WAV format for Whisper API
	wavBuffer := new(bytes.Buffer)
	// Write WAV header
	err = writeWAVHeader(wavBuffer, len(pcmBuffer), frameRate, channels, 16)
	if err != nil {
		return "", fmt.Errorf("failed to write WAV header: %w", err)
	}
	// Write PCM data
	for _, sample := range pcmBuffer {
		if err := binary.Write(wavBuffer, binary.LittleEndian, sample); err != nil {
			return "", fmt.Errorf("failed to write PCM data: %w", err)
		}
	}

	// Transcribe using OpenAI Whisper
	req := openai.AudioRequest{
		Model:    "whisper-1",
		Reader:   wavBuffer,
		FilePath: "audio.wav", // FilePath is required by the API, even though we're using Reader
	}
	resp, err := s.client.CreateTranscription(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to transcribe audio: %w", err)
	}

	log.Printf("🎤 Transcribed text: %s", resp.Text)
	return resp.Text, nil
}

// writeWAVHeader writes a WAV file header to the buffer
func writeWAVHeader(w *bytes.Buffer, numSamples, sampleRate, channels, bitsPerSample int) error {
	dataSize := numSamples * channels * (bitsPerSample / 8)
	fileSize := 36 + dataSize

	// RIFF header
	w.Write([]byte("RIFF"))
	if err := binary.Write(w, binary.LittleEndian, int32(fileSize)); err != nil {
		return err
	}
	w.Write([]byte("WAVE"))

	// fmt chunk
	w.Write([]byte("fmt "))
	if err := binary.Write(w, binary.LittleEndian, int32(16)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, int16(1)); err != nil { // PCM format
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, int16(channels)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, int32(sampleRate)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, int32(sampleRate*channels*bitsPerSample/8)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, int16(channels*bitsPerSample/8)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, int16(bitsPerSample)); err != nil {
		return err
	}

	// data chunk
	w.Write([]byte("data"))
	if err := binary.Write(w, binary.LittleEndian, int32(dataSize)); err != nil {
		return err
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
