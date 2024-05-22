package server

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

func homeRoutes(_ context.Context, r *gin.Engine) {
	//=========================
	// PAGES
	//=========================
	r.GET("/", func(c *gin.Context) {
		target := "index.html"

		d, e := strconv.Atoi(c.Query("d"))
		if e != nil {
			d = 1
		}

		regionsError := ""
		allClips, err := PersistenceService.RetrieveClipCount(-1)
		if err != nil {
			regionsError = err.Error()
		}

		p1Clips, err := PersistenceService.RetrieveClipCount(1)
		if err != nil {
			regionsError = err.Error()
		}

		p3Clips, err := PersistenceService.RetrieveClipCount(3)
		if err != nil {
			regionsError = err.Error()
		}

		regions, err := PersistenceService.RetrieveClipsStatsByRegion(d)
		if err != nil {
			regionsError = err.Error()
		}

		c.HTML(200, target, gin.H{
			"Tab":            "Home",
			"RegionsError":   regionsError,
			"Regions":        regions,
			"RegionDays":     fmt.Sprintf("Last %d days", d),
			"RegionCurrDays": d,
			"AllClipsCount":  allClips,
			"P1ClipsCount":   p1Clips,
			"P3ClipsCount":   p3Clips,
		})
	})

	r.GET("/clips", func(c *gin.Context) {
		target := "clips.html"
		if c.GetHeader("HX-Request") == "true" {
			target = "clips-list.html"
		}

		p, e := strconv.Atoi(c.Query("p"))
		if e != nil {
			p = 0
		}

		s, e := strconv.Atoi(c.Query("s"))
		if e != nil {
			s = DefaultPageSize
		}

		clipsError := ""
		clips, err := PersistenceService.RetrieveClipsByRegion(c.Query("t"), p, s)
		if err != nil {
			clipsError = err.Error()
		}

		c.HTML(200, target, gin.H{
			"Tab":           "Home",
			"ClipsRegion":   c.Query("t"),
			"ClipsError":    clipsError,
			"Clips":         clips,
			"ClipsPage":     p + 1,
			"ClipsPageSize": DefaultPageSize,
		})
	})

	//=========================
	// ACTIONS
	//=========================
	r.GET("/actions/load-more-clips", func(c *gin.Context) {
		c.Redirect(303, fmt.Sprintf("/clips?t=%s&p=%s&s=%s", c.Query("t"), c.Query("p"), c.Query("s")))
	})
}
