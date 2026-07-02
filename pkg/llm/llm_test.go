package llm

import (
	"context"
	"testing"
)

func TestMessageHelpers(t *testing.T) {
	t.Run("NewTextMessage", func(t *testing.T) {
		msg := NewTextMessage(RoleUser, "hello")
		if msg.Role != RoleUser {
			t.Fatalf("expected RoleUser, got %s", msg.Role)
		}
		if msg.Content != "hello" {
			t.Fatalf("expected 'hello', got '%s'", msg.Content)
		}
	})

	t.Run("NewSystemMessage", func(t *testing.T) {
		msg := NewSystemMessage("system prompt")
		if msg.Role != RoleSystem {
			t.Fatalf("expected RoleSystem, got %s", msg.Role)
		}
	})

	t.Run("NewUserMessage", func(t *testing.T) {
		msg := NewUserMessage("user query")
		if msg.Role != RoleUser || msg.Content != "user query" {
			t.Fatal("NewUserMessage failed")
		}
	})

	t.Run("NewAssistantMessage", func(t *testing.T) {
		msg := NewAssistantMessage("assistant reply")
		if msg.Role != RoleAssistant || msg.Content != "assistant reply" {
			t.Fatal("NewAssistantMessage failed")
		}
	})
}

func TestContentParts(t *testing.T) {
	t.Run("TextPart", func(t *testing.T) {
		part := TextPart("hello")
		if part.Type != ContentText || part.Text != "hello" {
			t.Fatal("TextPart failed")
		}
	})

	t.Run("ImagePart", func(t *testing.T) {
		part := ImagePart("data:image/png;base64,abc123")
		if part.Type != ContentImage || part.Base64 != "data:image/png;base64,abc123" {
			t.Fatal("ImagePart failed")
		}
	})

	t.Run("PDFPart", func(t *testing.T) {
		part := PDFPart("base64data")
		if part.Type != ContentPDF {
			t.Fatal("PDFPart type failed")
		}
		if part.Base64 != "data:application/pdf;base64,base64data" {
			t.Fatalf("PDFPart base64 failed: %s", part.Base64[:30])
		}
	})

	t.Run("NewMultiPartMessage", func(t *testing.T) {
		parts := []ContentPart{TextPart("text"), ImagePart("data:image/png;base64,img")}
		msg := NewMultiPartMessage(RoleUser, parts)
		if msg.Role != RoleUser {
			t.Fatal("MultiPartMessage role failed")
		}
		if len(msg.Parts) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(msg.Parts))
		}
	})
}

func TestParseBase64DataURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantMime    string
		wantData    string
		wantEmpty   bool
	}{
		{"valid image", "data:image/png;base64,abc123", "image/png", "abc123", false},
		{"valid pdf", "data:application/pdf;base64,xyz", "application/pdf", "xyz", false},
		{"no comma", "data:text/plain", "", "", true},
		{"empty string", "", "", "", true},
		{"short string", "abc", "", "", true},
		{"no mime", "data:;base64,data", "", "data", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mime, data := ParseBase64DataURL(tt.input)
			if tt.wantEmpty {
				if mime != "" || data != "" {
					t.Fatalf("expected empty, got mime=%s data=%s", mime, data)
				}
			} else {
				if mime != tt.wantMime {
					t.Fatalf("expected mime=%s, got %s", tt.wantMime, mime)
				}
				if data != tt.wantData {
					t.Fatalf("expected data=%s, got %s", tt.wantData, data)
				}
			}
		})
	}
}

func TestStreamChunk(t *testing.T) {
	chunk := StreamChunk{Type: ChunkContent, Content: "hello"}
	if chunk.Type != ChunkContent {
		t.Fatal("StreamChunk type failed")
	}
	if chunk.Content != "hello" {
		t.Fatal("StreamChunk content failed")
	}

	thinkChunk := StreamChunk{Type: ChunkThinking, Content: "thinking..."}
	if thinkChunk.Type != ChunkThinking {
		t.Fatal("Thinking chunk type failed")
	}
}

func TestProviderInterface(t *testing.T) {
	// Verify that a nil-safe provider can be used as interface
	var p Provider = &mockProvider{}
	if p == nil {
		t.Fatal("mockProvider should not be nil")
	}
}

// mockProvider implements Provider for testing
type mockProvider struct{}

func (m *mockProvider) GenerateContentStream(ctx context.Context, messages []Message, onChunk StreamCallback) (Message, error) {
	if onChunk != nil {
		for _, chunk := range []string{"hello", " ", "world"} {
			onChunk(StreamChunk{Type: ChunkContent, Content: chunk})
		}
	}
	return NewAssistantMessage("hello world"), nil
}

func (m *mockProvider) GenerateContent(ctx context.Context, model string, messages []Message) (Message, error) {
	return NewAssistantMessage("mock response"), nil
}

func (m *mockProvider) GetModels(ctx context.Context) ([]string, error) {
	return []string{"mock-model-v1", "mock-model-v2"}, nil
}

func (m *mockProvider) TestChat(ctx context.Context) error {
	return nil
}

func TestMockProvider(t *testing.T) {
	p := &mockProvider{}

	t.Run("GenerateContent", func(t *testing.T) {
		msg, err := p.GenerateContent(context.Background(), "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg.Content != "mock response" {
			t.Fatalf("expected 'mock response', got '%s'", msg.Content)
		}
	})

	t.Run("GenerateContentStream", func(t *testing.T) {
		var chunks []string
		msg, err := p.GenerateContentStream(context.Background(), nil, func(chunk StreamChunk) {
			chunks = append(chunks, chunk.Content)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg.Content != "hello world" {
			t.Fatalf("expected 'hello world', got '%s'", msg.Content)
		}
		if len(chunks) != 3 {
			t.Fatalf("expected 3 chunks, got %d", len(chunks))
		}
	})

	t.Run("GetModels", func(t *testing.T) {
		models, err := p.GetModels(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(models) != 2 {
			t.Fatalf("expected 2 models, got %d", len(models))
		}
	})

	t.Run("TestChat", func(t *testing.T) {
		err := p.TestChat(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
