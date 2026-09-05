package main

import (
	"log"
	"os"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/middleware"
	"github.com/fardilk/cms-porto-fardil/models"
	"github.com/fardilk/cms-porto-fardil/routes"
	"github.com/fardilk/cms-porto-fardil/seed"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.InitDB()

	config.DB.AutoMigrate(
		&models.User{},
		&models.Author{},
		&models.Category{},
		&models.Tag{},
		&models.Status{},
		&models.Article{},
		&models.MediaAsset{},
		&models.Service{},
		&models.ServiceHighlight{},
		&models.ServiceReason{},
		&models.ServiceStep{},
		&models.ServiceOutcome{},
		&models.ServiceMetric{},
		&models.ServiceFaq{},
		&models.ServicePlan{},
		&models.ServiceProof{},
		&models.ServiceSchedule{},
		&models.Lead{},
	)

	models.Migrate()

	// Statuses are reference data and safe to ensure on every boot.
	seed.SeedStatuses()

	// The rest is destructive: SeedCategories TRUNCATEs categories with CASCADE,
	// which also drops the article-category links, and SeedSampleData inserts a
	// fresh copy of the demo articles. Both used to run on every start, so any
	// restart wiped real data. They are opt-in now and refuse to run in prod.
	if os.Getenv("SEED_SAMPLE_DATA") == "true" {
		if config.IsProduction() {
			log.Fatal("SEED_SAMPLE_DATA is destructive and must not be used in production")
		}
		log.Println("WARNING: seeding sample data, this truncates categories")
		seed.SeedCategories()
		seed.SeedTags()
		seed.SeedSampleData()
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     config.AllowedOrigins(),
		AllowMethods:     []string{"POST", "GET", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	r.GET("/ping", func(c *gin.Context) { c.JSON(200, gin.H{"message": "pong"}) })
	r.GET("/healthz", routes.Health)

	r.POST("/login", routes.Login)
	r.GET("/dashboard", routes.Dashboard)
	r.GET("/me", routes.Me)

	// Public reads. The site build and the CMS both use these.
	public := r.Group("/api")
	{
		public.GET("/categories", routes.GetCategories)
		public.GET("/categories/:id", routes.GetCategory)
		public.GET("/articles", routes.GetArticles)
		public.GET("/articles/:id", routes.GetArticle)
		public.GET("/tags", routes.GetTags)
		public.GET("/tags/:id", routes.GetTag)
		public.GET("/services", routes.GetServices)
		public.GET("/services/:id", routes.GetService)
		public.GET("/media", routes.ListMedia)
		public.GET("/media/usage", routes.MediaUsage)

		// The only unauthenticated write: the public contact form.
		public.POST("/leads", routes.CreateLead)
	}

	// Writes. These were wide open: anyone who could reach the API could create,
	// edit or delete articles, categories and tags.
	authed := r.Group("/api", middleware.RequireAuth())
	{
		authed.POST("/categories", routes.CreateCategory)
		authed.PUT("/categories/:id", routes.UpdateCategory)
		authed.DELETE("/categories/:id", routes.DeleteCategory)

		authed.POST("/articles", routes.CreateArticle)
		authed.PUT("/articles/:id", routes.UpdateArticle)
		authed.PATCH("/articles/:id", routes.PatchArticle)
		authed.DELETE("/articles/:id", routes.DeleteArticle)

		authed.POST("/tags", routes.CreateTag)

		authed.POST("/services", routes.CreateService)
		authed.PUT("/services/:id", routes.UpdateService)
		authed.PATCH("/services/:id", routes.PatchService)
		authed.DELETE("/services/:id", routes.DeleteService)

		authed.GET("/stats", routes.GetStats)
		authed.GET("/leads", routes.GetLeads)
		authed.PATCH("/leads/:id", routes.UpdateLead)
		authed.DELETE("/leads/:id", routes.DeleteLead)

		authed.POST("/media", routes.UploadMedia)
		authed.DELETE("/media/:id", routes.DeleteMedia)
	}

	r.Static("/images", config.UploadDir())

	log.Printf("listening on %s (env=%s)", config.Port(), config.AppEnv())
	if err := r.Run(config.Port()); err != nil {
		log.Fatal(err)
	}
}
