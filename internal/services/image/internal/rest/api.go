package rest

import (
	"io"
	"net/http"
	"net/url"

	"github.com/gamma-omg/lexi-go/internal/pkg/httpx"
	"github.com/gamma-omg/lexi-go/internal/pkg/serr"
)

type imageService interface {
	Upload(img io.Reader) (*url.URL, error)
}

type mux interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	Handle(pattern string, handler http.Handler)
}

type API struct {
	srv         imageService
	maxImgSize  int64
	contentRoot string
}

func NewAPI(srv imageService, maxImgSize int64, contentRoot string) *API {
	return &API{
		srv:         srv,
		maxImgSize:  maxImgSize,
		contentRoot: contentRoot,
	}
}

func (api *API) Register(m mux) {
	fs := http.FileServer(http.Dir(api.contentRoot))

	m.HandleFunc("POST /upload", api.handleUploadImage)
	m.Handle("GET /image/", http.StripPrefix("/image/", fs))
}

type uploadImageResponse struct {
	ImageURL string `json:"image_url"`
}

func (api *API) handleUploadImage(w http.ResponseWriter, r *http.Request) {
	f, _, err := r.FormFile("image")
	if err != nil {
		httpx.HandleErr(w, r, serr.NewServiceError(err, http.StatusBadRequest, "invalid image"))
		return
	}
	defer f.Close()

	img := http.MaxBytesReader(w, f, api.maxImgSize)
	imgUrl, err := api.srv.Upload(img)
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}

	httpx.WriteResponse(w, http.StatusCreated, uploadImageResponse{
		ImageURL: imgUrl.String(),
	})
}
