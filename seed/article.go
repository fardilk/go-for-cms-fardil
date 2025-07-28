package seed

import (
	"time"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
	"gorm.io/datatypes"
)

func SeedSampleData() {
	config.DB.AutoMigrate(&models.Status{}, &models.Article{}) // Migrate first

	// Ensure statuses exist
	statuses := []string{"DRAFT", "PUBLISHED", "ARCHIVED", "DELETED"}
	for _, name := range statuses {
		config.DB.FirstOrCreate(&models.Status{}, models.Status{Name: name})
	}

	// Now fetch the status IDs
	var statusPublished, statusDraft models.Status
	config.DB.Where("name = ?", "PUBLISHED").First(&statusPublished)
	config.DB.Where("name = ?", "DRAFT").First(&statusDraft)

	// Ensure at least one author, category, and tag exist
	var author models.Author
	config.DB.FirstOrCreate(&author, models.Author{Name: "Admin"})
	var category models.Category
	config.DB.FirstOrCreate(&category, models.Category{Name: "Programming", Slug: "programming"})
	if category.ID == 0 {
		panic("Category not found or created!")
	}

	var tag models.Tag
	config.DB.FirstOrCreate(&tag, models.Tag{Name: "Go", Slug: "go"})

	// Wipe out previous articles
	config.DB.Exec("TRUNCATE TABLE articles RESTART IDENTITY CASCADE;")

	articles := []models.Article{
		{
			Title:           "Go for Beginners",
			Slug:            "go-for-beginners",
			MetaTitle:       "Go for Beginners",
			MetaDescription: "A beginner's guide to Go programming.",
			FeaturedImage:   "/static/images/img-1.png",
			AltText:         "Go for Beginners image",
			Excerpt:         "Start learning Go from scratch.",
			CanonicalURL:    "https://example.com/go-for-beginners",
			ReadingTime:     4,
			Content:         datatypes.JSON([]byte(`{"blocks":[{"type":"paragraph","data":{"text":"Welcome to Go programming for beginners."}}]}`)),
			PublishedAt:     ptrTime(time.Now()),
			UpdatedAt:       ptrTime(time.Now()),
			IsFeatured:      true,
			IsDeleted:       false,
			Authors:         []models.Author{author},
			Categories:      []models.Category{category},
			Tags:            []models.Tag{tag},
			StatusID:        statusPublished.ID, // This sets status_id
		},
		{
			Title:           "Go Advanced Patterns",
			Slug:            "go-advanced-patterns",
			MetaTitle:       "Go Advanced Patterns",
			MetaDescription: "Explore advanced patterns in Go.",
			FeaturedImage:   "/static/images/img-2.png",
			AltText:         "Go Advanced Patterns image",
			Excerpt:         "Take your Go skills to the next level.",
			CanonicalURL:    "https://example.com/go-advanced-patterns",
			ReadingTime:     6,
			Content:         datatypes.JSON([]byte(`{"blocks":[{"type":"paragraph","data":{"text":"Advanced Go programming patterns and practices."}}]}`)),
			PublishedAt:     ptrTime(time.Now()),
			UpdatedAt:       ptrTime(time.Now()),
			IsFeatured:      false,
			IsDeleted:       false,
			Authors:         []models.Author{author},
			Categories:      []models.Category{category},
			Tags:            []models.Tag{tag},
			StatusID:        statusDraft.ID,
		},
		{
			Title:           "Business Strategy",
			Slug:            "business-strategy",
			MetaTitle:       "Business Strategy",
			MetaDescription: "Effective strategies for business growth.",
			FeaturedImage:   "/static/images/img-3.png",
			AltText:         "Business Strategy image",
			Excerpt:         "Grow your business with proven strategies.",
			CanonicalURL:    "https://example.com/business-strategy",
			ReadingTime:     5,
			Content:         datatypes.JSON([]byte(`{"blocks":[{"type":"paragraph","data":{"text":"Business strategies for success."}}]}`)),
			PublishedAt:     ptrTime(time.Now()),
			UpdatedAt:       ptrTime(time.Now()),
			IsFeatured:      false,
			IsDeleted:       false,
			Authors:         []models.Author{author},
			Categories:      []models.Category{category},
			Tags:            []models.Tag{tag},
			StatusID:        statusPublished.ID,
		},
		{
			Title:           "Finance Tips",
			Slug:            "finance-tips",
			MetaTitle:       "Finance Tips",
			MetaDescription: "Tips for managing your finances.",
			FeaturedImage:   "/static/images/img-4.png",
			AltText:         "Finance Tips image",
			Excerpt:         "Manage your money wisely.",
			CanonicalURL:    "https://example.com/finance-tips",
			ReadingTime:     3,
			Content:         datatypes.JSON([]byte(`{"blocks":[{"type":"paragraph","data":{"text":"Finance tips for everyone."}}]}`)),
			PublishedAt:     ptrTime(time.Now()),
			UpdatedAt:       ptrTime(time.Now()),
			IsFeatured:      false,
			IsDeleted:       false,
			Authors:         []models.Author{author},
			Categories:      []models.Category{category},
			Tags:            []models.Tag{tag},
			StatusID:        statusPublished.ID,
		},
		{
			Title:           "Health & Wellness",
			Slug:            "health-wellness",
			MetaTitle:       "Health & Wellness",
			MetaDescription: "Improve your health and wellness.",
			FeaturedImage:   "/static/images/img-5.png",
			AltText:         "Health & Wellness image",
			Excerpt:         "Tips for a healthier life.",
			CanonicalURL:    "https://example.com/health-wellness",
			ReadingTime:     4,
			Content:         datatypes.JSON([]byte(`{"blocks":[{"type":"paragraph","data":{"text":"Health and wellness tips."}}]}`)),
			PublishedAt:     ptrTime(time.Now()),
			UpdatedAt:       ptrTime(time.Now()),
			IsFeatured:      false,
			IsDeleted:       false,
			Authors:         []models.Author{author},
			Categories:      []models.Category{category},
			Tags:            []models.Tag{tag},
			StatusID:        statusDraft.ID,
		},
		{
			Title:           "Teamwork in Action",
			Slug:            "teamwork-in-action",
			MetaTitle:       "Teamwork in Action",
			MetaDescription: "How teamwork drives success.",
			FeaturedImage:   "/static/images/img-6.png",
			AltText:         "Teamwork image",
			Excerpt:         "The power of working together.",
			CanonicalURL:    "https://example.com/teamwork-in-action",
			ReadingTime:     5,
			Content:         datatypes.JSON([]byte(`{"blocks":[{"type":"paragraph","data":{"text":"Teamwork makes the dream work."}}]}`)),
			PublishedAt:     ptrTime(time.Now()),
			UpdatedAt:       ptrTime(time.Now()),
			IsFeatured:      false,
			IsDeleted:       false,
			Authors:         []models.Author{author},
			Categories:      []models.Category{category},
			Tags:            []models.Tag{tag},
			StatusID:        statusPublished.ID,
		},
		{
			Title:           "Creative Planning",
			Slug:            "creative-planning",
			MetaTitle:       "Creative Planning",
			MetaDescription: "Plan creatively for better results.",
			FeaturedImage:   "/static/images/img-7.png",
			AltText:         "Creative Planning image",
			Excerpt:         "Unlock your creative potential.",
			CanonicalURL:    "https://example.com/creative-planning",
			ReadingTime:     4,
			Content:         datatypes.JSON([]byte(`{"blocks":[{"type":"paragraph","data":{"text":"Creative planning for success."}}]}`)),
			PublishedAt:     ptrTime(time.Now()),
			UpdatedAt:       ptrTime(time.Now()),
			IsFeatured:      false,
			IsDeleted:       false,
			Authors:         []models.Author{author},
			Categories:      []models.Category{category},
			Tags:            []models.Tag{tag},
			StatusID:        statusDraft.ID,
		},
		{
			Title:           "Digital Innovation",
			Slug:            "digital-innovation",
			MetaTitle:       "Digital Innovation",
			MetaDescription: "Innovate in the digital age.",
			FeaturedImage:   "/static/images/img-8.png",
			AltText:         "Digital Innovation image",
			Excerpt:         "Embrace digital transformation.",
			CanonicalURL:    "https://example.com/digital-innovation",
			ReadingTime:     5,
			Content:         datatypes.JSON([]byte(`{"blocks":[{"type":"paragraph","data":{"text":"Digital innovation for modern business."}}]}`)),
			PublishedAt:     ptrTime(time.Now()),
			UpdatedAt:       ptrTime(time.Now()),
			IsFeatured:      false,
			IsDeleted:       false,
			Authors:         []models.Author{author},
			Categories:      []models.Category{category},
			Tags:            []models.Tag{tag},
			StatusID:        statusPublished.ID,
		},
	}

	for _, article := range articles {
		config.DB.Create(&article)
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
