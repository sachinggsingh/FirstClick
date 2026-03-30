package main

import (
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sachinggsingh/firstclick/internal/logger"
	"github.com/sachinggsingh/firstclick/internal/model"
	"github.com/sachinggsingh/firstclick/internal/realtime"
	redisconn "github.com/sachinggsingh/firstclick/internal/redis"
	"github.com/sachinggsingh/firstclick/internal/service"
	"github.com/sachinggsingh/firstclick/internal/store"
	"github.com/sachinggsingh/firstclick/internal/utils"
)

var movies = []movieResponse{
	{ID: "inception", Title: "Inception", Rows: 5, SeatsPerRow: 8},
	{ID: "dune", Title: "Dune: Part Two", Rows: 4, SeatsPerRow: 6},
}

type movieResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Rows        int    `json:"rows"`
	SeatsPerRow int    `json:"seats_per_row"`
}

type seatStatusResponse struct {
	SeatID    string `json:"seat_id"`
	Booked    bool   `json:"booked"`
	Confirmed bool   `json:"confirmed"`
	UserID    string `json:"user_id"`
}

type holdRequest struct {
	UserID string `json:"user_id"`
}

type userIDRequest struct {
	UserID string `json:"user_id"`
}

func main() {
	logger.InitGlobalLogger()
	router := gin.Default()

	staticDir := "./static"
	indexPath := filepath.Join(staticDir, "index.html")
	router.Static("/static", staticDir)

	seatsHub := realtime.NewSeatsHub()
	router.GET("/ws/seats", func(c *gin.Context) {
		seatsHub.HandleWS(c.Writer, c.Request)
	})

	// Serve index.html for SPA routes / and any unknown GET route.
	router.GET("/", func(c *gin.Context) {
		c.File(indexPath)
	})
	router.GET("/index.html", func(c *gin.Context) {
		c.File(indexPath)
	})

	// --- Redis + services ---
	var bookingSvc *service.BookingService
	redisClient, err := redisconn.ConnectToRedis()
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
	} else {
		rdb := store.NewRedisStore(redisClient)
		bookingSvc = service.NewBookingService(rdb, nil)
	}

	// --- APIs used by static/index.html ---
	router.GET("/movies", func(c *gin.Context) {
		utils.WriteJSON(c.Writer, http.StatusOK, movies)
	})

	router.GET("/movies/:movieId/seats", func(c *gin.Context) {
		if bookingSvc == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "booking service not available"})
			return
		}

		movieID := c.Param("movieId")
		bookings, err := bookingSvc.GetBookedSeats(c.Request.Context(), movieID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp := make([]seatStatusResponse, 0, len(bookings))
		for _, b := range bookings {
			resp = append(resp, seatStatusResponse{
				SeatID:    b.SeatID,
				Booked:    true,
				Confirmed: b.Status == model.SeatConfirm,
				UserID:    b.UserID,
			})
		}
		utils.WriteJSON(c.Writer, http.StatusOK, resp)
	})

	router.POST("/movies/:movieId/seats/:seatId/hold", func(c *gin.Context) {
		if bookingSvc == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "booking service not available"})
			return
		}

		var req holdRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.UserID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}

		movieID := c.Param("movieId")
		seatID := c.Param("seatId")

		booking := model.Booking{
			MovieID: movieID,
			SeatID:  seatID,
			UserID:  req.UserID,
		}

		_, err := bookingSvc.Book(c.Request.Context(), &booking)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		utils.WriteJSON(c.Writer, http.StatusOK, gin.H{
			"session_id": booking.ID,
			"movie_id":   booking.MovieID,
			"seat_id":    booking.SeatID,
			"expires_at": booking.ExpiresAt.UnixMilli(),
		})

		// Notify all connected clients to refresh this movie's seat grid.
		seatsHub.BroadcastSeatsUpdated(movieID)
	})

	router.PUT("/sessions/:sessionId/confirm", func(c *gin.Context) {
		if bookingSvc == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "booking service not available"})
			return
		}

		var req userIDRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.UserID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}

		sessionID := c.Param("sessionId")
		confirmed, err := bookingSvc.Confirm(c.Request.Context(), sessionID, req.UserID)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent) // 204

		seatsHub.BroadcastSeatsUpdated(confirmed.MovieID)
	})

	router.DELETE("/sessions/:sessionId", func(c *gin.Context) {
		if bookingSvc == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "booking service not available"})
			return
		}

		var req userIDRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.UserID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}

		sessionID := c.Param("sessionId")
		// We don't return booking from Release(), but we can broadcast safely by
		// refreshing all movies. To keep it simple we broadcast with a best-effort:
		// the store already knows movieID; here we just broadcast a generic update
		// by refreshing the currently selected movie on the client side.
		if err := bookingSvc.Release(c.Request.Context(), sessionID, req.UserID); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent) // 204

		// Seat released; we may not know which movie the seat belongs to here,
		// so notify clients to refresh their currently selected movie.
		seatsHub.BroadcastSeatsUpdated("")
	})

	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			c.File(indexPath)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	slog.Info("Server is running on port 8080", "port", "8080")
	if err := router.Run(":8080"); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}
