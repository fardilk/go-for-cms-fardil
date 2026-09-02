package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
)

// countPair is one row of a grouped count, e.g. leads per status.
type countPair struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type upcomingBatch struct {
	Service    string     `json:"service"`
	Slug       string     `json:"slug"`
	Category   string     `json:"category"`
	StartsAt   *time.Time `json:"starts_at"`
	City       string     `json:"city"`
	Format     string     `json:"format"`
	SeatsTotal int        `json:"seats_total"`
	SeatsLeft  int        `json:"seats_left"`
	Leads      int64      `json:"leads"`
}

type statsResponse struct {
	Leads struct {
		Total    int64       `json:"total"`
		New      int64       `json:"new"`
		Last7    int64       `json:"last_7_days"`
		Last30   int64       `json:"last_30_days"`
		ByStatus []countPair `json:"by_status"`
		ByDay    []countPair `json:"by_day"`
		ByPage   []countPair `json:"by_page"`
	} `json:"leads"`

	Content struct {
		ArticlesPublished int64 `json:"articles_published"`
		ArticlesDraft     int64 `json:"articles_draft"`
		ServicesPublished int64 `json:"services_published"`
		ServicesDraft     int64 `json:"services_draft"`
		Media             int64 `json:"media"`
	} `json:"content"`

	Upcoming []upcomingBatch `json:"upcoming"`
}

// jakarta is where the business and every enquiry is, so a "day" is a day
// there rather than wherever the server happens to run.
var jakarta = time.FixedZone("WIB", 7*60*60)

const trendDays = 30

func leadsByDay(db *gorm.DB, now time.Time) []countPair {
	var stamps []time.Time
	db.Model(&models.Lead{}).
		Where("created_at >= ?", now.AddDate(0, 0, -trendDays)).
		Order("created_at").Pluck("created_at", &stamps)

	counts := make(map[string]int64, trendDays)
	for _, t := range stamps {
		counts[t.In(jakarta).Format("2006-01-02")]++
	}

	days := make([]countPair, 0, trendDays)
	start := now.In(jakarta).AddDate(0, 0, -trendDays+1)
	for i := 0; i < trendDays; i++ {
		key := start.AddDate(0, 0, i).Format("2006-01-02")
		days = append(days, countPair{Key: key, Count: counts[key]})
	}
	return days
}

// GetStats handles GET /api/stats: the numbers the dashboard reports on.
//
// Every figure is counted from the tables at request time. Nothing is cached
// and nothing is precomputed, because the data set is small enough that a
// stale number would cost more than the query does.
func GetStats(c *gin.Context) {
	var out statsResponse
	db := config.DB
	now := time.Now()

	db.Model(&models.Lead{}).Count(&out.Leads.Total)
	db.Model(&models.Lead{}).Where("status = ?", models.LeadNew).Count(&out.Leads.New)
	db.Model(&models.Lead{}).Where("created_at >= ?", now.AddDate(0, 0, -7)).Count(&out.Leads.Last7)
	db.Model(&models.Lead{}).Where("created_at >= ?", now.AddDate(0, 0, -30)).Count(&out.Leads.Last30)

	out.Leads.ByStatus = []countPair{}
	db.Model(&models.Lead{}).
		Select("status AS key, COUNT(*) AS count").
		Group("status").Order("count DESC").Scan(&out.Leads.ByStatus)

	// A row per day for the last 30, including the empty ones so a chart does
	// not silently close the gaps. Bucketed in Go rather than in SQL: date
	// formatting and time zones differ per driver, and thirty days of one small
	// column is nothing to read.
	out.Leads.ByDay = leadsByDay(db, now)

	// Which page produced the enquiry. source_path carries the query string, so
	// a programme link (?program=slug) is distinguishable from a plain visit.
	out.Leads.ByPage = []countPair{}
	db.Model(&models.Lead{}).
		Select("COALESCE(NULLIF(source_path, ''), '(tidak diketahui)') AS key, COUNT(*) AS count").
		Group("COALESCE(NULLIF(source_path, ''), '(tidak diketahui)')").
		Order("count DESC").Limit(15).Scan(&out.Leads.ByPage)

	// Same definition of "published" the public listing uses, so the dashboard
	// count matches what the site actually shows.
	pub := db.Model(&models.Article{}).
		Where("status_id = ?", statusID("PUBLISHED")).
		Where("published_at IS NOT NULL")
	pub.Count(&out.Content.ArticlesPublished)

	var articles int64
	db.Model(&models.Article{}).Count(&articles)
	out.Content.ArticlesDraft = articles - out.Content.ArticlesPublished
	db.Model(&models.Service{}).Where("published = ?", true).Count(&out.Content.ServicesPublished)
	db.Model(&models.Service{}).Where("published = ?", false).Count(&out.Content.ServicesDraft)
	db.Model(&models.MediaAsset{}).Count(&out.Content.Media)

	out.Upcoming = []upcomingBatch{}
	db.Table("service_schedules AS sc").
		Select(`s.title AS service, s.slug AS slug, s.category AS category,
		        sc.starts_at, sc.city, sc.format, sc.seats_total, sc.seats_left,
		        (SELECT COUNT(*) FROM leads l WHERE l.source_path LIKE '%program=' || s.slug) AS leads`).
		Joins("JOIN services s ON s.id = sc.service_id").
		Where("sc.starts_at IS NOT NULL AND sc.starts_at >= ?", now).
		Order("sc.starts_at").Limit(10).Scan(&out.Upcoming)

	c.JSON(http.StatusOK, out)
}
