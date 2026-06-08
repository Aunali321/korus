package handlers

import (
	"database/sql"

	"github.com/Aunali321/korus/internal/services"
	aisvc "github.com/Aunali321/korus/internal/services/ai"
)

type Handler struct {
	db                *sql.DB
	dbPath            string
	auth              *services.AuthService
	scanner           *services.ScannerService
	search            *services.SearchService
	transcoder        *services.Transcoder
	musicBrainz       *services.MusicBrainzService
	listenBrainz      *services.ListenBrainzService
	ai                *aisvc.Service
	mediaRoot         string
	radioDefaultLimit int
}

func New(db *sql.DB, dbPath string, auth *services.AuthService, scanner *services.ScannerService, search *services.SearchService, transcoder *services.Transcoder, mb *services.MusicBrainzService, lb *services.ListenBrainzService, aiSvc *aisvc.Service, mediaRoot string, radioDefaultLimit int) *Handler {
	return &Handler{
		db:                db,
		dbPath:            dbPath,
		auth:              auth,
		scanner:           scanner,
		search:            search,
		transcoder:        transcoder,
		musicBrainz:       mb,
		listenBrainz:      lb,
		ai:                aiSvc,
		mediaRoot:         mediaRoot,
		radioDefaultLimit: radioDefaultLimit,
	}
}
