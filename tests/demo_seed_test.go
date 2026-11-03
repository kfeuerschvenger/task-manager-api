package tests

import (
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/kfeuerschvenger/task-manager-api/database"
	"github.com/kfeuerschvenger/task-manager-api/models"
	"github.com/kfeuerschvenger/task-manager-api/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const demoTestPassword = "test12345"

// newDemoSet returns a demo set that belongs to one test: unique emails and a fresh namespace, so its user and
// task ids collide with nothing, including the real demo rows. Its users and tasks are deleted when the test ends.
func newDemoSet(t *testing.T) services.DemoSet {
	t.Helper()

	set := services.DefaultDemoSet(services.DefaultDemoReference)
	set.Creator.Email = uniqueEmail("demo-creator")
	set.Assignee.Email = uniqueEmail("demo-assignee")
	set.Namespace = uuid.New()

	t.Cleanup(func() { deleteDemoSetRows(t, set) })
	return set
}

// deleteDemoSetRows removes the tasks and the users the set derives. It only ever matches the set's own ids.
func deleteDemoSetRows(t *testing.T, set services.DemoSet) {
	t.Helper()

	creatorID, assigneeID := set.UserIDs()
	ids := []uuid.UUID{creatorID, assigneeID}
	assert.NoError(t, database.DB.Where("creator_id IN ? OR assignee_id IN ?", ids, ids).Delete(&models.Task{}).Error)
	assert.NoError(t, database.DB.Where("id IN ?", ids).Delete(&models.User{}).Error)
}

// demoSnapshot reads the set's users and the tasks they created, in a stable order.
func demoSnapshot(t *testing.T, set services.DemoSet) ([]models.User, []models.Task) {
	t.Helper()
	return snapshotByEmail(t, []string{set.Creator.Email, set.Assignee.Email})
}

// snapshotByEmail reads the users with these emails and the tasks they created, in a stable order.
func snapshotByEmail(t *testing.T, emails []string) ([]models.User, []models.Task) {
	t.Helper()

	var users []models.User
	require.NoError(t, database.DB.Where("email IN ?", emails).Order("email").Find(&users).Error)

	ids := make([]string, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID.String())
	}
	var tasks []models.Task
	require.NoError(t, database.DB.Where("creator_id IN ?", ids).Order("id").Find(&tasks).Error)
	return users, tasks
}

func TestSeedDemoIsIdempotent(t *testing.T) {
	set := newDemoSet(t)

	first, err := services.SeedDemoSet(set, demoTestPassword)
	require.NoError(t, err)
	usersBefore, tasksBefore := demoSnapshot(t, set)

	second, err := services.SeedDemoSet(set, demoTestPassword)
	require.NoError(t, err)
	usersAfter, tasksAfter := demoSnapshot(t, set)

	assert.Equal(t, first, second)
	assert.Len(t, usersAfter, 2)
	assert.Len(t, tasksAfter, first.Tasks)
	// Reading the rows back after the second run gives the same values as after the first: no user or task is
	// duplicated, the user rows keep their ids, hashes and timestamps, and the tasks come back with the same ids
	// and fields. The tasks are deleted and inserted again, so this does not claim they were never rewritten.
	assert.Equal(t, usersBefore, usersAfter)
	assert.Equal(t, tasksBefore, tasksAfter)
}

func TestSeedDemoResetsDemoTasksOnly(t *testing.T) {
	set := newDemoSet(t)

	_, err := services.SeedDemoSet(set, demoTestPassword)
	require.NoError(t, err)
	_, tasks := demoSnapshot(t, set)
	require.NotEmpty(t, tasks)

	// Drift: a demo task is edited and an extra one is added by the demo creator.
	require.NoError(t, database.DB.Model(&models.Task{}).Where("id = ?", tasks[0].ID).Update("status", "complete").Error)
	extra := models.Task{Title: "Left behind", Description: "Added after the seed", DueDate: tasks[0].DueDate,
		CreatorID: tasks[0].CreatorID, AssigneeID: tasks[0].CreatorID}
	require.NoError(t, database.DB.Create(&extra).Error)

	// An unrelated user with a task must survive the reset.
	other := newTestUser(t)
	otherTask := createTestTask(t, other.Token, "low", "pending")

	_, err = services.SeedDemoSet(set, demoTestPassword)
	require.NoError(t, err)

	_, after := demoSnapshot(t, set)
	assert.Equal(t, tasks, after)

	resp := doAuthed(t, http.MethodGet, "/tasks/"+otherTask, other.Token, nil)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestSeedDemoUsersCanLogIn(t *testing.T) {
	set := newDemoSet(t)

	_, err := services.SeedDemoSet(set, demoTestPassword)
	require.NoError(t, err)

	for _, email := range []string{set.Creator.Email, set.Assignee.Email} {
		code, _ := postJSON(t, "/auth/login", map[string]string{"email": email, "password": demoTestPassword})
		assert.Equal(t, http.StatusOK, code, email)
	}
}

func TestSeedDemoSetLeavesTheRealDemoAlone(t *testing.T) {
	realEmails := []string{services.DemoCreatorEmail, services.DemoAssigneeEmail}
	usersBefore, tasksBefore := snapshotByEmail(t, realEmails)

	_, err := services.SeedDemoSet(newDemoSet(t), demoTestPassword)
	require.NoError(t, err)

	usersAfter, tasksAfter := snapshotByEmail(t, realEmails)
	assert.Equal(t, usersBefore, usersAfter)
	assert.Equal(t, tasksBefore, tasksAfter)
}

func TestSeedDemoRefusesAUserWithADifferentID(t *testing.T) {
	set := newDemoSet(t)

	// A user that already holds the assignee email under an id the seed does not derive, with a task of its own.
	stranger := models.User{ID: uuid.New(), FirstName: "Someone", LastName: "Else", Email: set.Assignee.Email, Password: "kept-as-is"}
	require.NoError(t, database.DB.Create(&stranger).Error)
	task := models.Task{Title: "Not part of the demo", Description: "Owned by the stranger", DueDate: services.DefaultDemoReference,
		Priority: "low", Status: "pending", CreatorID: stranger.ID, AssigneeID: stranger.ID}
	require.NoError(t, database.DB.Create(&task).Error)
	// The stranger is outside the set's derived ids, so the set's own cleanup does not reach it.
	t.Cleanup(func() {
		assert.NoError(t, database.DB.Where("id = ?", task.ID).Delete(&models.Task{}).Error)
		assert.NoError(t, database.DB.Where("id = ?", stranger.ID).Delete(&models.User{}).Error)
	})

	summary, err := services.SeedDemoSet(set, demoTestPassword)

	require.Error(t, err)
	assert.True(t, errors.Is(err, services.ErrDemoUserConflict), err.Error())
	assert.Equal(t, services.DemoSummary{}, summary)

	// The stranger and its task are unchanged, and the run rolled back, so the creator was not created either.
	var userAfter models.User
	require.NoError(t, database.DB.First(&userAfter, "id = ?", stranger.ID).Error)
	assert.Equal(t, stranger.Password, userAfter.Password)
	assert.Equal(t, stranger.FirstName, userAfter.FirstName)

	var taskAfter models.Task
	require.NoError(t, database.DB.First(&taskAfter, "id = ?", task.ID).Error)
	assert.Equal(t, task.Title, taskAfter.Title)

	var creators int64
	require.NoError(t, database.DB.Model(&models.User{}).Where("email = ?", set.Creator.Email).Count(&creators).Error)
	assert.Zero(t, creators)
}
