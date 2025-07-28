package seed

import (
	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
)

func SeedTags() {
	config.DB.Exec("TRUNCATE TABLE tags RESTART IDENTITY CASCADE;")

	tags := []string{
		"Go", "Backend", "Frontend", "DevOps", "Cloud", "Database",
		"API", "Security", "Testing", "Performance", "Design", "UX",
	}

	for _, name := range tags {
		slug := generateSlug(name)
		config.DB.Create(&models.Tag{Name: name, Slug: slug})
	}
}

