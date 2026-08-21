package handlers

import (
	"database/sql"

	"github.com/Aunali321/korus/internal/services"
	aisvc "github.com/Aunali321/korus/internal/services/ai"
)

type Deps struct {
	DB                *sql.DB
	DBPath            string
	Auth              *services.AuthService
	Scanner           *services.ScannerService
	Search            *services.SearchService
	Library           *services.LibraryService
	Playlists         *services.PlaylistService
	Stats             *services.StatsService
	Transcoder        *services.Transcoder
	MusicBrainz       *services.MusicBrainzService
	ListenBrainz      *services.ListenBrainzService
	AI                *aisvc.Service
	MediaRoot         string
	RadioDefaultLimit int
}

type Handler struct {
	db                *sql.DB
	dbPath            string
	auth              *services.AuthService
	scanner           *services.ScannerService
	search            *services.SearchService
	library           *services.LibraryService
	playlists         *services.PlaylistService
	stats             *services.StatsService
	transcoder        *services.Transcoder
	musicBrainz       *services.MusicBrainzService
	listenBrainz      *services.ListenBrainzService
	ai                *aisvc.Service
	mediaRoot         string
	radioDefaultLimit int
}

func New(deps Deps) *Handler {
	return &Handler{
		db:                deps.DB,
		dbPath:            deps.DBPath,
		auth:              deps.Auth,
		scanner:           deps.Scanner,
		search:            deps.Search,
		library:           deps.Library,
		playlists:         deps.Playlists,
		stats:             deps.Stats,
		transcoder:        deps.Transcoder,
		musicBrainz:       deps.MusicBrainz,
		listenBrainz:      deps.ListenBrainz,
		ai:                deps.AI,
		mediaRoot:         deps.MediaRoot,
		radioDefaultLimit: deps.RadioDefaultLimit,
	}
}
