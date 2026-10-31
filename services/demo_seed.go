package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kfeuerschvenger/task-manager-api/database"
	"github.com/kfeuerschvenger/task-manager-api/models"
	"github.com/kfeuerschvenger/task-manager-api/utils"
	"gorm.io/gorm"
)

const (
	// DemoCreatorEmail is the email of the demo user who creates every demo task.
	DemoCreatorEmail = "maya.rivera@example.com"
	// DemoAssigneeEmail is the email of the demo user who receives some of the demo tasks.
	DemoAssigneeEmail = "theo.nakamura@example.com"
)

// demoNamespace derives the fixed identifiers of the demo rows, so the same seed always produces the same ids.
var demoNamespace = uuid.MustParse("6f1d2c3a-5b7e-4a90-8d21-0c4e9a7b3f10")

// demoReference is the instant every demo due date is measured from. It is a constant so the data does not
// drift with the day the seed runs.
var demoReference = time.Date(2026, time.November, 2, 9, 0, 0, 0, time.UTC)

// DemoSummary reports what SeedDemo wrote.
type DemoSummary struct {
	Users int
	Tasks int
}

// demoTask describes one demo task. Owner and Assignee are the emails of demo users, and DueInDays is relative
// to demoReference.
type demoTask struct {
	Slug        string
	Title       string
	Description string
	Priority    string
	Status      string
	DueInDays   int
	Assignee    string
}

// demoTasks is the demo content. Every task is created by DemoCreatorEmail. The titles stay sober, with a
// wink only around deadlines and the status lifecycle.
func demoTasks() []demoTask {
	return []demoTask{
		{"readme", "Write the README before the due date", "Cover what the API does, how to run it and how to call it. Finish while the due date is still in the future.", "high", "in_progress", 2, DemoCreatorEmail},
		{"promote", "Promote a task from pending to in_progress", "Pick the oldest pending task and move it forward. Staring at it does not count as progress.", "medium", "pending", 5, DemoAssigneeEmail},
		{"review", "Review the pull request promised on Friday", "Read the diff, leave comments that explain why, and approve only what you would deploy yourself.", "high", "pending", 1, DemoAssigneeEmail},
		{"release", "Ship the release candidate (mostly on time)", "Tag the build, publish the notes and tell the team where the changelog lives.", "high", "in_progress", 4, DemoCreatorEmail},
		{"secret", "Rotate the JWT secret on the agreed date", "Generate a new 32 byte secret, deploy it and expect every active session to end.", "high", "pending", 9, DemoCreatorEmail},
		{"first", "Mark the first task complete", "Moving a task to complete is the part of the lifecycle everyone enjoys.", "low", "complete", -3, DemoCreatorEmail},
		{"tests", "Add a test for the case that works on my machine", "Reproduce the failure in a clean environment first, then write the test that keeps it fixed.", "medium", "in_progress", 6, DemoAssigneeEmail},
		{"gofmt", "Run gofmt and let the tabs win", "Format every package before merging. The formatter has the final word.", "low", "complete", -1, DemoAssigneeEmail},
		{"retro", "Plan the retrospective for the sprint that has not ended", "Draft the agenda early. Deadlines have a habit of arriving before the agenda does.", "medium", "pending", 12, DemoCreatorEmail},
		{"backup", "Back up the database before the deadline, not after", "Run the backup, restore it into a scratch database and check a few rows.", "medium", "pending", 7, DemoCreatorEmail},
	}
}

// demoIdentity builds the user rows for the two demo accounts. The id is derived from the email.
func demoIdentity() []models.User {
	return []models.User{
		{ID: uuid.NewSHA1(demoNamespace, []byte(DemoCreatorEmail)), FirstName: "Maya", LastName: "Rivera", Email: DemoCreatorEmail},
		{ID: uuid.NewSHA1(demoNamespace, []byte(DemoAssigneeEmail)), FirstName: "Theo", LastName: "Nakamura", Email: DemoAssigneeEmail},
	}
}

// SeedDemo writes the deterministic demo data: two users and their tasks. It is idempotent. Users are matched
// by email and never duplicated, and the tasks created by the demo users are replaced by the same set with the
// same ids, so a second run leaves the rows as the first one did. Users and tasks of anyone else are not touched.
//
// The caller must have validated the password and refused production.
func SeedDemo(password string) (DemoSummary, error) {
	users := demoIdentity()
	ids := make(map[string]uuid.UUID, len(users))

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, user := range users {
			id, err := upsertDemoUser(tx, user, password)
			if err != nil {
				return err
			}
			ids[user.Email] = id
		}

		creatorID := ids[DemoCreatorEmail]
		demoIDs := []uuid.UUID{creatorID, ids[DemoAssigneeEmail]}
		if err := tx.Where("creator_id IN ?", demoIDs).Delete(&models.Task{}).Error; err != nil {
			return fmt.Errorf("clearing demo tasks: %w", err)
		}

		tasks := demoTasks()
		rows := make([]models.Task, 0, len(tasks))
		for _, task := range tasks {
			due := demoReference.AddDate(0, 0, task.DueInDays)
			rows = append(rows, models.Task{
				ID:          uuid.NewSHA1(demoNamespace, []byte("task:"+task.Slug)),
				Title:       task.Title,
				Description: task.Description,
				DueDate:     due,
				Priority:    task.Priority,
				Status:      task.Status,
				CreatorID:   creatorID,
				AssigneeID:  ids[task.Assignee],
				CreatedAt:   demoReference,
				UpdatedAt:   demoReference,
			})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("creating demo tasks: %w", err)
		}
		return nil
	})
	if err != nil {
		return DemoSummary{}, err
	}

	return DemoSummary{Users: len(users), Tasks: len(demoTasks())}, nil
}

// upsertDemoUser makes sure the demo user exists with the given name and password and returns its id. An
// existing row keeps its id, and its hash is only replaced when the password no longer matches, so re-running
// the seed with the same password leaves the row untouched.
func upsertDemoUser(tx *gorm.DB, user models.User, password string) (uuid.UUID, error) {
	// Find with a limit instead of First: a missing row is the normal first run and must not log as an error.
	var found []models.User
	if err := tx.Where("email = ?", user.Email).Limit(1).Find(&found).Error; err != nil {
		return uuid.Nil, fmt.Errorf("looking up %s: %w", user.Email, err)
	}

	if len(found) == 0 {
		hash, err := utils.HashPassword(password)
		if err != nil {
			return uuid.Nil, fmt.Errorf("hashing demo password: %w", err)
		}
		user.Password = hash
		user.CreatedAt = demoReference
		user.UpdatedAt = demoReference
		if err := tx.Create(&user).Error; err != nil {
			return uuid.Nil, fmt.Errorf("creating %s: %w", user.Email, err)
		}
		return user.ID, nil
	}

	existing := found[0]
	updates := map[string]interface{}{}
	if existing.FirstName != user.FirstName {
		updates["first_name"] = user.FirstName
	}
	if existing.LastName != user.LastName {
		updates["last_name"] = user.LastName
	}
	if !utils.CheckPasswordHash(password, existing.Password) {
		hash, err := utils.HashPassword(password)
		if err != nil {
			return uuid.Nil, fmt.Errorf("hashing demo password: %w", err)
		}
		updates["password"] = hash
	}
	if len(updates) > 0 {
		if err := tx.Model(&models.User{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			return uuid.Nil, fmt.Errorf("updating %s: %w", user.Email, err)
		}
	}
	return existing.ID, nil
}
