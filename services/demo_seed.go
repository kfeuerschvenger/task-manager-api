package services

import (
	"errors"
	"fmt"
	"strings"
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

// DemoReferenceVariable names the optional environment variable that moves the reference date of the demo
// due dates. It holds a date in YYYY-MM-DD format.
const DemoReferenceVariable = "DEMO_REFERENCE"

// demoNamespace derives the fixed identifiers of the demo rows, so the same seed always produces the same ids.
var demoNamespace = uuid.MustParse("6f1d2c3a-5b7e-4a90-8d21-0c4e9a7b3f10")

// DefaultDemoReference is the instant every demo due date is measured from unless DEMO_REFERENCE overrides it.
// It is fixed so the data does not drift with the day the seed runs.
var DefaultDemoReference = time.Date(2026, time.November, 2, 9, 0, 0, 0, time.UTC)

// ErrDemoUserConflict is returned when a user with a demo email exists under an id the seed did not derive.
var ErrDemoUserConflict = errors.New("a user with the demo email exists under a different id")

// ParseDemoReference turns the value of DEMO_REFERENCE into the reference instant. An empty value selects
// DefaultDemoReference, and anything that is not a valid YYYY-MM-DD date is rejected.
func ParseDemoReference(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultDemoReference, nil
	}
	day, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be a date in YYYY-MM-DD format, got %q", DemoReferenceVariable, value)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), 9, 0, 0, 0, time.UTC), nil
}

// DemoUser is the identity of one demo account.
type DemoUser struct {
	Email     string
	FirstName string
	LastName  string
}

// DemoSet is everything that identifies one seeded demo: the two accounts, the namespace the ids are derived
// from, and the reference instant of the due dates. The real demo is DefaultDemoSet. A test passes its own set,
// with a fresh namespace and unique emails, so it never touches the real demo rows.
type DemoSet struct {
	Creator   DemoUser
	Assignee  DemoUser
	Namespace uuid.UUID
	Reference time.Time
}

// DefaultDemoSet is the real demo: Maya and Theo, the fixed namespace and the given reference instant.
func DefaultDemoSet(reference time.Time) DemoSet {
	return DemoSet{
		Creator:   DemoUser{Email: DemoCreatorEmail, FirstName: "Maya", LastName: "Rivera"},
		Assignee:  DemoUser{Email: DemoAssigneeEmail, FirstName: "Theo", LastName: "Nakamura"},
		Namespace: demoNamespace,
		Reference: reference,
	}
}

// UserIDs returns the ids the set derives for its creator and assignee.
func (set DemoSet) UserIDs() (creator, assignee uuid.UUID) {
	return uuid.NewSHA1(set.Namespace, []byte(set.Creator.Email)), uuid.NewSHA1(set.Namespace, []byte(set.Assignee.Email))
}

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

// demoIdentity builds the user rows for the two accounts of the set. The id is derived from the email.
func demoIdentity(set DemoSet) []models.User {
	creatorID, assigneeID := set.UserIDs()
	return []models.User{
		{ID: creatorID, FirstName: set.Creator.FirstName, LastName: set.Creator.LastName, Email: set.Creator.Email},
		{ID: assigneeID, FirstName: set.Assignee.FirstName, LastName: set.Assignee.LastName, Email: set.Assignee.Email},
	}
}

// SeedDemo writes the real demo data, measured from the given reference instant. See SeedDemoSet.
func SeedDemo(password string, reference time.Time) (DemoSummary, error) {
	return SeedDemoSet(DefaultDemoSet(reference), password)
}

// SeedDemoSet writes the deterministic demo data of one set: two users and their tasks. It is idempotent. A user
// is matched by email and only updated when its id is the one the set derives, and the tasks created by the set's
// users are replaced by the same tasks with the same ids, so a second run leaves the rows as the first one did.
// A user that holds a demo email under any other id aborts the run with ErrDemoUserConflict, and neither that
// user nor its tasks are touched. Users and tasks of anyone else are not touched either.
//
// The caller must have validated the password and refused production.
func SeedDemoSet(set DemoSet, password string) (DemoSummary, error) {
	users := demoIdentity(set)
	ids := make(map[string]uuid.UUID, len(users))

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, user := range users {
			id, err := upsertDemoUser(tx, user, password, set.Reference)
			if err != nil {
				return err
			}
			ids[user.Email] = id
		}

		creatorID := ids[set.Creator.Email]
		assigneeID := ids[set.Assignee.Email]
		// Both ids equal the derived ones here, so the deletion cannot reach a row the set does not own.
		if err := tx.Where("creator_id IN ?", []uuid.UUID{creatorID, assigneeID}).Delete(&models.Task{}).Error; err != nil {
			return fmt.Errorf("clearing demo tasks: %w", err)
		}

		tasks := demoTasks()
		rows := make([]models.Task, 0, len(tasks))
		for _, task := range tasks {
			due := set.Reference.AddDate(0, 0, task.DueInDays)
			// The task table names its assignee by the real demo email, which stands for the set's assignee.
			taskAssignee := creatorID
			if task.Assignee == DemoAssigneeEmail {
				taskAssignee = assigneeID
			}
			rows = append(rows, models.Task{
				ID:          uuid.NewSHA1(set.Namespace, []byte("task:"+task.Slug)),
				Title:       task.Title,
				Description: task.Description,
				DueDate:     due,
				Priority:    task.Priority,
				Status:      task.Status,
				CreatorID:   creatorID,
				AssigneeID:  taskAssignee,
				CreatedAt:   set.Reference,
				UpdatedAt:   set.Reference,
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

// upsertDemoUser makes sure the demo user exists with the given name and password and returns its id. A row
// found by email must carry the id derived for the user, otherwise the call fails with ErrDemoUserConflict
// before anything is written. A matching row keeps its id, and its hash is only replaced when the password no
// longer matches, so re-running the seed with the same password leaves the row untouched.
func upsertDemoUser(tx *gorm.DB, user models.User, password string, reference time.Time) (uuid.UUID, error) {
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
		user.CreatedAt = reference
		user.UpdatedAt = reference
		if err := tx.Create(&user).Error; err != nil {
			return uuid.Nil, fmt.Errorf("creating %s: %w", user.Email, err)
		}
		return user.ID, nil
	}

	existing := found[0]
	if existing.ID != user.ID {
		return uuid.Nil, fmt.Errorf("%w: %s has id %s, the demo derives %s; remove or rename that user before seeding",
			ErrDemoUserConflict, user.Email, existing.ID, user.ID)
	}
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
