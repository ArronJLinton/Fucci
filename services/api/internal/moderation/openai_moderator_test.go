package moderation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMediaPreviewURL_PhotoPassthrough(t *testing.T) {
	t.Parallel()
	u := "https://res.cloudinary.com/demo/image/upload/v1/story.jpg"
	assert.Equal(t, u, MediaPreviewURL(u, "photo"))
}

func TestMediaPreviewURL_VideoFrame(t *testing.T) {
	t.Parallel()
	in := "https://res.cloudinary.com/demo/video/upload/v123/clip.mp4"
	got := MediaPreviewURL(in, "video")
	assert.Contains(t, got, "/upload/so_0/")
	assert.True(t, strings.HasSuffix(got, ".jpg"))
}

func TestOpenAIModerator_CheckFlagged(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/moderations", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"mod-1","model":"omni-moderation-latest",
			"results":[{"flagged":true,"categories":{"hate":true,"violence":false},"category_scores":{}}]
		}`))
	}))
	defer srv.Close()

	m := NewOpenAIModerator("test-key", srv.URL)
	m.Client = srv.Client()
	res, err := m.Check(context.Background(), ContentCheckInput{Texts: []string{"bad"}})
	require.NoError(t, err)
	assert.True(t, res.Flagged)
	assert.Contains(t, res.Categories, "hate")
}

func TestOpenAIModerator_EmptyInput(t *testing.T) {
	t.Parallel()
	m := NewOpenAIModerator("test-key", "http://example.invalid")
	res, err := m.Check(context.Background(), ContentCheckInput{})
	require.NoError(t, err)
	assert.False(t, res.Flagged)
}

func TestFormatAIFlagDescription(t *testing.T) {
	t.Parallel()
	d := FormatAIFlagDescription(ContentCheckResult{Model: "omni", Categories: []string{"hate"}})
	assert.Contains(t, d, "hate")
	assert.Contains(t, d, "soft-flag")
}
