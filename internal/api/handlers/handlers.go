package handlers

import (
	"database/sql"

	"github.com/Aunali321/korus/internal/services"
)

type Handler struct {
	db                *sql.DB
	auth              *services.AuthService
	scanner           *services.ScannerService
	search            *services.SearchService
	transcoder        *services.Transcoder
	musicBrainz       *services.MusicBrainzService
	listenBrainz      *services.ListenBrainzService
	radio             *services.RadioService
	mediaRoot         string
	radioDefaultLimit int
}

func New(db *sql.DB, auth *services.AuthService, scanner *services.ScannerService, search *services.SearchService, transcoder *services.Transcoder, mb *services.MusicBrainzService, lb *services.ListenBrainzService, radio *services.RadioService, mediaRoot string, radioDefaultLimit int) *Handler {
	return &Handler{
		db:                db,
		auth:              auth,
		scanner:           scanner,
		search:            search,
		transcoder:        transcoder,
		musicBrainz:       mb,
		listenBrainz:      lb,
		radio:             radio,
		mediaRoot:         mediaRoot,
		radioDefaultLimit: radioDefaultLimit,
	}
}
