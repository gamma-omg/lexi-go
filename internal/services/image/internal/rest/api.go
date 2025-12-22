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

type API struct {
	srv         imageService
	maxImgSize  int64
	contentRoot string
	mux         *http.ServeMux
}

func NewAPI(srv imageService, maxImgSize int64, contentRoot string) *API {
	api := &API{
		srv:         srv,
		maxImgSize:  maxImgSize,
		contentRoot: contentRoot,
		mux:         http.NewServeMux(),
	}

	api.mount()
	return api
}

func (api *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.mux.ServeHTTP(w, r)
}

func (api *API) mount() {
	fs := http.FileServer(http.Dir(api.contentRoot))

	api.mux.HandleFunc("POST /upload", api.handleUploadImage)
	api.mux.Handle("GET /image/", http.StripPrefix("/image/", fs))
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

	err = httpx.WriteJSON(w, http.StatusCreated, uploadImageResponse{
		ImageURL: imgUrl.String(),
	})
	if err != nil {
		httpx.HandleErr(w, r, err)
		return
	}
}
