package handlers

import (
	"database/sql"

	"github.com/Aunali321/korus/internal/services"
)

type Handler struct {
	db           *sql.DB
	auth         *services.AuthService
	scanner      *services.ScannerService
	search       *services.SearchService
	transcoder   *services.Transcoder
	musicBrainz  *services.MusicBrainzService
	listenBrainz *services.ListenBrainzService
	mediaRoot    string
}

func New(db *sql.DB, auth *services.AuthService, scanner *services.ScannerService, search *services.SearchService, transcoder *services.Transcoder, mb *services.MusicBrainzService, lb *services.ListenBrainzService, mediaRoot string) *Handler {
	return &Handler{
		db:           db,
		auth:         auth,
		scanner:      scanner,
		search:       search,
		transcoder:   transcoder,
		musicBrainz:  mb,
		listenBrainz: lb,
		mediaRoot:    mediaRoot,
	}
}
