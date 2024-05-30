package v20

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/imartingraham/facebook-marketing-api-golang-sdk/fb"
	"golang.org/x/sync/errgroup"
)

// VideoService works with advideos.
type VideoService struct {
	c *fb.Client
}

// Get returns a single Video.
func (vs *VideoService) Get(ctx context.Context, id string) (*Video, error) {
	res := &Video{}
	err := vs.c.GetJSON(ctx, fb.NewRoute(Version, "/%s", id).Fields(advideoFields...).String(), res)
	if err != nil {
		if fb.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return res, nil
}

// Upload uploads a video from r into an account.
func (vs *VideoService) Upload(ctx context.Context, act, title string, size int64, r io.Reader) (*Video, error) {
	url := fb.NewRoute(Version, "/act_%s/advideos", act).String()

	res := uploadVideoResponse{}
	err := vs.c.PostJSON(ctx, url, uploadVideoRequestStart{
		UploadPhase: "start",
		FileSize:    size,
	}, &res)
	if err != nil {
		return nil, err
	}

	for size > 0 {
		chunksize := res.EndOffset - res.StartOffset
		if chunksize > size {
			chunksize = size
		}
		size -= chunksize
		err := vs.c.UploadFile(ctx, url, title, io.LimitReader(r, chunksize), map[string]string{
			"upload_phase":      "transfer",
			"upload_session_id": res.UploadSessionID,
			"start_offset":      fmt.Sprintf("%d", res.StartOffset),
		}, &res)
		if err != nil {
			return nil, err
		}
	}

	fr := finishResponse{}
	err = vs.c.PostJSON(ctx, url, uploadVideoRequestEnd{
		UploadPhase:     "finish",
		UploadSessionID: res.UploadSessionID,
		Title:           title,
	}, &fr)
	if err != nil {
		return nil, err
	}

	return vs.Get(ctx, res.VideoID)
}

// ReadList returns all videos from an account and writes them to a channel.
func (vs *VideoService) ReadList(ctx context.Context, act string, res chan<- Video) error {
	jres := make(chan json.RawMessage)
	wg := errgroup.Group{}
	wg.Go(func() error {
		defer close(jres)

		return vs.c.ReadList(ctx, fb.NewRoute(Version, "/act_%s/advideos", act).Fields(advideoFields...).Limit(1000).String(), jres)
	})
	wg.Go(func() error {
		for e := range jres {
			v := Video{}
			err := json.Unmarshal(e, &v)
			if err != nil {
				return err
			}
			res <- v
		}

		return nil
	})

	return wg.Wait()
}

var advideoFields = []string{"title", "id", "picture", "description", "from", "format", "length", "status"}

type uploadVideoRequestStart struct {
	UploadPhase string `json:"upload_phase"`
	FileSize    int64  `json:"file_size"`
}

type uploadVideoRequestEnd struct {
	UploadPhase     string `json:"upload_phase"`
	UploadSessionID string `json:"upload_session_id"`
	Title           string `json:"title"`
}

type uploadVideoResponse struct {
	UploadSessionID string `json:"upload_session_id"`
	VideoID         string `json:"video_id"`
	StartOffset     int64  `json:"start_offset,string"`
	EndOffset       int64  `json:"end_offset,string"`
}

type finishResponse struct {
	Success bool `json:"success"`
}

// Video represents an ad video.
type Video struct {
	ContentCategory        string  `json:"content_category"`
	CreatedTime            string  `json:"created_time"`
	Description            string  `json:"description"`
	EmbedHTML              string  `json:"embed_html"`
	Embeddable             bool    `json:"embeddable"`
	ID                     string  `json:"id"`
	Icon                   string  `json:"icon"`
	Length                 float64 `json:"length"`
	MonetizationStatus     string  `json:"monetization_status"`
	Picture                string  `json:"picture"`
	IsCrosspostVideo       bool    `json:"is_crosspost_video"`
	IsCrosspostingEligible bool    `json:"is_crossposting_eligible"`
	IsInstagramEligible    bool    `json:"is_instagram_eligible"`
	PermalinkURL           string  `json:"permalink_url"`
	Published              bool    `json:"published"`
	Source                 string  `json:"source"`
	UpdatedTime            string  `json:"updated_time"`
	Title                  string  `json:"title,omitempty"`
	AutoGeneratedCaptions  struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Paging struct {
			Cursors struct {
				Before string `json:"before"`
				After  string `json:"after"`
			} `json:"cursors"`
		} `json:"paging"`
	} `json:"auto_generated_captions,omitempty"`
	Format []struct {
		EmbedHTML string `json:"embed_html"`
		Filter    string `json:"filter"`
		Height    int    `json:"height"`
		Picture   string `json:"picture"`
		Width     int    `json:"width"`
	} `json:"format"`
	From struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"from"`
	Privacy struct {
		Allow       string `json:"allow"`
		Deny        string `json:"deny"`
		Description string `json:"description"`
		Friends     string `json:"friends"`
		Networks    string `json:"networks"`
		Value       string `json:"value"`
	} `json:"privacy"`
	Status struct {
		VideoStatus string `json:"video_status"`
	} `json:"status"`
}
