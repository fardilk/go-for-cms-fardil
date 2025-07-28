package seed

import (
	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
)

func SeedStatuses() {
	statuses := []string{"DRAFT", "PUBLISHED", "ARCHIVED", "DELETED"}
	for _, name := range statuses {
		config.DB.FirstOrCreate(&models.Status{}, models.Status{Name: name})
	}
}
