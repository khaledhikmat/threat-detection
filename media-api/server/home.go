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
			d = 24 * 60
		}

		regionsError := ""
		allClips, err := PersistenceService.RetrieveClipCount(24 * 60)
		if err != nil {
			regionsError = err.Error()
		}

		p1Clips, err := PersistenceService.RetrieveClipCount(1 * 60)
		if err != nil {
			regionsError = err.Error()
		}

		p3Clips, err := PersistenceService.RetrieveClipCount(3 * 60)
		if err != nil {
			regionsError = err.Error()
		}

		regions, err := PersistenceService.RetrieveClipsStatsByRegion(d)
		if err != nil {
			regionsError = err.Error()
		}

		c.HTML(200, target, gin.H{
			"Tab":               "Home",
			"RegionsError":      regionsError,
			"Regions":           regions,
			"RegionPeriods":     fmt.Sprintf("Last %d minutes", d),
			"RegionCurrPeriods": d,
			"AllClipsCount":     allClips,
			"P1ClipsCount":      p1Clips,
			"P3ClipsCount":      p3Clips,
		})
	})

	r.GET("/clips", func(c *gin.Context) {
		target := "clips.html"
		if c.GetHeader("HX-Request") == "true" {
			target = "clips-list.html"
		}

		d, e := strconv.Atoi(c.Query("d"))
		if e != nil {
			d = 24 * 60
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
		clips, err := PersistenceService.RetrieveClipsByRegion(c.Query("t"), d, p, s)
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

	r.GET("/alerts", func(c *gin.Context) {
		target := "alerts.html"
		if c.GetHeader("HX-Request") == "true" {
			target = "alerts-list.html"
		}

		t, e := strconv.Atoi(c.Query("t"))
		if e != nil {
			t = DefaultPageSize
		}

		d, e := strconv.Atoi(c.Query("d"))
		if e != nil {
			d = 24 * 60
		}

		clipsError := ""
		clips, err := PersistenceService.RetrieveAlertedClips(t, d)
		if err != nil {
			clipsError = err.Error()
		}

		c.HTML(200, target, gin.H{
			"Tab":         "Home",
			"ClipsError":  clipsError,
			"Clips":       clips,
			"ClipsTop":    t,
			"ClipsPeriod": d,
		})
	})

	r.GET("/clip", func(c *gin.Context) {
		fmt.Printf("***** 🎥 clip id: %s\n", c.Query("id"))
		target := "clip.html"
		if c.Query("id") == "" {
			c.HTML(200, target, gin.H{
				"Tab":   "Home",
				"Error": "Campaign id is missing!",
			})
			return
		}

		clip, err := PersistenceService.RetrieveClipByID(c.Query("id"))
		if err != nil {
			c.HTML(200, target, gin.H{
				"Tab":   "Home",
				"Error": err.Error(),
			})
			return
		}

		fmt.Printf("***** 🎥 clip id return: %s\n", clip.ID)
		c.HTML(200, target, gin.H{
			"Tab":   "Home",
			"Error": "",
			"Clip":  clip,
		})
	})

	//=========================
	// ACTIONS
	//=========================
	r.GET("/actions/load-more-clips", func(c *gin.Context) {
		c.Redirect(303, fmt.Sprintf("/clips?t=%s&p=%s&s=%s", c.Query("t"), c.Query("p"), c.Query("s")))
	})
}
