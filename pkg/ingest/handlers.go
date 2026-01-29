package ingest

import (
	"net/http"

	"rag-system/internal/ingest"

	"github.com/gin-gonic/gin"
)

type IngestHandler struct {
	service *ingest.Service
}

func NewIngestHandler(s *ingest.Service) *IngestHandler {
	return &IngestHandler{service: s}
}

func (h *IngestHandler) IngestPDF(c *gin.Context) {
	topic := c.PostForm("topic")
	if topic == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "topic is required"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pdf file is required"})
		return
	}
	defer file.Close()

	err = h.service.IngestPDFAsync(c.Request.Context(), file, header.Filename, topic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status": "accepted",
		"file":   header.Filename,
		"topic":  topic,
	})
}
