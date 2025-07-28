package seed

import (
	"strings"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
)

func SeedCategories() {
	// Clear all previous records and reset increment (Postgres)
	config.DB.Exec("TRUNCATE TABLE categories RESTART IDENTITY CASCADE;")

	categories := []string{
		"Technology", "Health & Wellness", "Business", "Finance", "Education",
		"Lifestyle", "Travel", "Food & Drink", "Fashion", "Entertainment",
		"Sports", "Science", "Environment", "Parenting", "Personal Development",
	}

	for _, name := range categories {
		slug := generateSlug(name)
		config.DB.Create(&models.Category{Name: name, Slug: slug})
	}
}

// Simple slug generator
func generateSlug(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}
