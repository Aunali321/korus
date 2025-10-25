package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"korus/internal/config"
	"korus/internal/middleware"
	"korus/internal/services"
)

type RecommendHandler struct {
	recommender *services.RecommenderService
	cfg         *config.RecommenderConfig
}

func NewRecommendHandler(recommender *services.RecommenderService, cfg *config.RecommenderConfig) *RecommendHandler {
	if cfg == nil {
		cfg = &config.RecommenderConfig{SimilarityLimit: 50}
	}
	return &RecommendHandler{recommender: recommender, cfg: cfg}
}

func (h *RecommendHandler) enabled() bool {
	return h.recommender != nil && h.recommender.Enabled()
}

func (h *RecommendHandler) featureDisabled(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"error":   "feature_disabled",
		"message": "Recommendations feature is disabled",
	})
}

func (h *RecommendHandler) SimilarSongs(c *gin.Context) {
	if !h.enabled() {
		h.featureDisabled(c)
		return
	}

	songID, err := strconv.Atoi(c.Param("id"))
	if err != nil || songID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid song ID",
		})
		return
	}

	limit := h.parseLimit(c.Query("limit"))

	songs, err := h.recommender.SimilarSongs(c.Request.Context(), songID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to generate similar songs",
		})
		return
	}

	c.JSON(http.StatusOK, songs)
}

func (h *RecommendHandler) UserRecommendations(c *gin.Context) {
	if !h.enabled() {
		h.featureDisabled(c)
		return
	}

	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	limit := h.parseLimit(c.Query("limit"))
	songs, err := h.recommender.UserRecommendations(c.Request.Context(), user.ID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to generate recommendations",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"songs": songs,
	})
}

type radioRequest struct {
	Seeds struct {
		Songs []int `json:"songs"`
	} `json:"seeds"`
	Limit *int `json:"limit,omitempty"`
}

func (h *RecommendHandler) Radio(c *gin.Context) {
	if !h.enabled() {
		h.featureDisabled(c)
		return
	}

	var req radioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	limit := h.cfg.SimilarityLimit
	if req.Limit != nil && *req.Limit > 0 {
		limit = h.normLimit(*req.Limit)
	}

	songs, err := h.recommender.Radio(c.Request.Context(), req.Seeds.Songs, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to generate radio playlist",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"songs": songs,
	})
}

func (h *RecommendHandler) parseLimit(limitParam string) int {
	if limitParam == "" {
		return h.cfg.SimilarityLimit
	}
	parsed, err := strconv.Atoi(limitParam)
	if err != nil || parsed <= 0 {
		return h.cfg.SimilarityLimit
	}
	return h.normLimit(parsed)
}

func (h *RecommendHandler) normLimit(limit int) int {
	if limit <= 0 {
		return h.cfg.SimilarityLimit
	}
	if limit > h.cfg.SimilarityLimit {
		return h.cfg.SimilarityLimit
	}
	return limit
}
