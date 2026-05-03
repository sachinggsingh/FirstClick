package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func closeResponseBody(resp *http.Response) {
	if err := resp.Body.Close(); err != nil {
		fmt.Println("Error in closing the resp.Body", err)
	}
}
func TestMoviesEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupTestRouter()

	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/movies")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var movies []movieResponse
	if err := json.NewDecoder(resp.Body).Decode(&movies); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	expectedMovies := []movieResponse{
		{ID: "inception", Title: "Inception", Rows: 5, SeatsPerRow: 8},
		{ID: "dune", Title: "Dune: Part Two", Rows: 4, SeatsPerRow: 6},
	}

	if len(movies) != len(expectedMovies) {
		t.Errorf("Expected %d movies, got %d", len(expectedMovies), len(movies))
	}

	for i, expected := range expectedMovies {
		if i >= len(movies) {
			break
		}
		actual := movies[i]
		if actual.ID != expected.ID || actual.Title != expected.Title ||
			actual.Rows != expected.Rows || actual.SeatsPerRow != expected.SeatsPerRow {
			t.Errorf("Movie %d: expected %+v, got %+v", i, expected, actual)
		}
	}
}

func TestConfigEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupTestRouter()
	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/config.js")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/javascript; charset=utf-8" {
		t.Errorf("Expected content type 'application/javascript; charset=utf-8', got '%s'", contentType)
	}
}

func setupTestRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/movies", func(c *gin.Context) {
		if err := json.NewEncoder(c.Writer).Encode(movies); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode response"})
			return
		}
	})

	router.GET("/config.js", func(c *gin.Context) {
		c.Header("Content-Type", "application/javascript; charset=utf-8")
		c.String(http.StatusOK, "window.APP_CONFIG = { backendOrigin: %q };", "http://localhost:8080")
	})

	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			c.String(http.StatusOK, "SPA fallback")
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	return router
}
