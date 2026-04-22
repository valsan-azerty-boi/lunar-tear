package service

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"lunar-tear/server/internal/schedule"
)

// RegisterAdminRoutes creates the admin HTTP mux for serving the admin UI and REST API.
func RegisterAdminRoutes(manager *schedule.Manager) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/admin/", handleAdminUI)
	mux.HandleFunc("/admin/api/status", handleAdminStatus(manager))
	mux.HandleFunc("/admin/api/schedule", handleAdminSchedule(manager))
	mux.HandleFunc("/admin/api/bundles", handleAdminBundles(manager))
	mux.HandleFunc("/admin/api/reload", handleAdminReload(manager))

	log.Printf("[admin] registered admin routes on /admin/")
	return mux
}

func handleAdminUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(adminHTML))
}

func handleAdminStatus(manager *schedule.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		stats := manager.Stats()
		sched := manager.Schedule()

		resp := struct {
			Stats    schedule.ScheduleStats `json:"stats"`
			Schedule schedule.ContentSchedule `json:"schedule"`
		}{
			Stats:    stats,
			Schedule: sched,
		}

		writeJSON(w, resp)
	}
}

func handleAdminSchedule(manager *schedule.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			sched := manager.Schedule()
			writeJSON(w, sched)

		case http.MethodPost:
			var sched schedule.ContentSchedule
			if err := json.NewDecoder(r.Body).Decode(&sched); err != nil {
				http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
				return
			}

			if err := manager.UpdateSchedule(&sched); err != nil {
				http.Error(w, "failed to update schedule: "+err.Error(), http.StatusInternalServerError)
				return
			}

			stats := manager.Stats()
			resp := struct {
				OK    bool                   `json:"ok"`
				Stats schedule.ScheduleStats `json:"stats"`
			}{
				OK:    true,
				Stats: stats,
			}
			writeJSON(w, resp)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleAdminBundles(manager *schedule.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		idx := manager.BundleIndex()

		// Build sorted month list
		months := make([]string, 0, len(idx.Bundles))
		for m := range idx.Bundles {
			months = append(months, m)
		}
		sort.Strings(months)

		type bundleInfo struct {
			Month         string  `json:"month"`
			EventCount    int     `json:"event_count"`
			GachaCount    int     `json:"gacha_count"`
			LoginCount    int     `json:"login_count"`
			SideStoryCount int   `json:"side_story_count"`
			EventChapters []int32 `json:"event_chapters"`
			GachaIds      []int32 `json:"gacha_ids"`
			LoginBonuses  []int32 `json:"login_bonuses"`
			SideStories   []int32 `json:"side_stories"`
		}

		bundles := make([]bundleInfo, 0, len(months))
		for _, m := range months {
			b := idx.Bundles[m]
			bundles = append(bundles, bundleInfo{
				Month:          m,
				EventCount:     len(b.EventChapters),
				GachaCount:     len(b.GachaIds),
				LoginCount:     len(b.LoginBonuses),
				SideStoryCount: len(b.SideStories),
				EventChapters:  b.EventChapters,
				GachaIds:       b.GachaIds,
				LoginBonuses:   b.LoginBonuses,
				SideStories:    b.SideStories,
			})
		}

		resp := struct {
			Bundles    []bundleInfo     `json:"bundles"`
			Permanent  *schedule.Bundle `json:"permanent"`
			Unreleased *schedule.Bundle `json:"unreleased"`
		}{
			Bundles:    bundles,
			Permanent:  idx.Permanent,
			Unreleased: idx.Unreleased,
		}

		writeJSON(w, resp)
	}
}

func handleAdminReload(manager *schedule.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := manager.Reload(); err != nil {
			http.Error(w, "reload failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		stats := manager.Stats()
		resp := struct {
			OK    bool                   `json:"ok"`
			Stats schedule.ScheduleStats `json:"stats"`
		}{
			OK:    true,
			Stats: stats,
		}
		writeJSON(w, resp)
	}
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		log.Printf("[admin] JSON encode error: %v", err)
	}
}
