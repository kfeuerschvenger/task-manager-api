package tests

import (
	"net/http"
	"testing"

	"github.com/kfeuerschvenger/task-manager-api/database"
	"github.com/kfeuerschvenger/task-manager-api/models"
	"github.com/kfeuerschvenger/task-manager-api/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const demoTestPassword = "test12345"

// demoSnapshot reads the demo users and the tasks they created, in a stable order.
func demoSnapshot(t *testing.T) ([]models.User, []models.Task) {
	t.Helper()

	var users []models.User
	require.NoError(t, database.DB.
		Where("email IN ?", []string{services.DemoCreatorEmail, services.DemoAssigneeEmail}).
		Order("email").Find(&users).Error)

	ids := make([]string, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID.String())
	}
	var tasks []models.Task
	require.NoError(t, database.DB.Where("creator_id IN ?", ids).Order("id").Find(&tasks).Error)
	return users, tasks
}

func TestSeedDemoIsIdempotent(t *testing.T) {
	first, err := services.SeedDemo(demoTestPassword)
	require.NoError(t, err)
	usersBefore, tasksBefore := demoSnapshot(t)

	second, err := services.SeedDemo(demoTestPassword)
	require.NoError(t, err)
	usersAfter, tasksAfter := demoSnapshot(t)

	assert.Equal(t, first, second)
	assert.Len(t, usersAfter, 2)
	assert.Len(t, tasksAfter, first.Tasks)
	// The ids, hashes and every task field are identical: the second run neither duplicates nor rewrites a row.
	assert.Equal(t, usersBefore, usersAfter)
	assert.Equal(t, tasksBefore, tasksAfter)
}

func TestSeedDemoResetsDemoTasksOnly(t *testing.T) {
	_, err := services.SeedDemo(demoTestPassword)
	require.NoError(t, err)
	_, tasks := demoSnapshot(t)
	require.NotEmpty(t, tasks)

	// Drift: a demo task is edited and an extra one is added by the demo creator.
	require.NoError(t, database.DB.Model(&models.Task{}).Where("id = ?", tasks[0].ID).Update("status", "complete").Error)
	extra := models.Task{Title: "Left behind", Description: "Added after the seed", DueDate: tasks[0].DueDate,
		CreatorID: tasks[0].CreatorID, AssigneeID: tasks[0].CreatorID}
	require.NoError(t, database.DB.Create(&extra).Error)

	// An unrelated user with a task must survive the reset.
	other := newTestUser(t)
	otherTask := createTestTask(t, other.Token, "low", "pending")

	_, err = services.SeedDemo(demoTestPassword)
	require.NoError(t, err)

	_, after := demoSnapshot(t)
	assert.Equal(t, tasks, after)

	resp := doAuthed(t, http.MethodGet, "/tasks/"+otherTask, other.Token, nil)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestSeedDemoUsersCanLogIn(t *testing.T) {
	_, err := services.SeedDemo(demoTestPassword)
	require.NoError(t, err)

	for _, email := range []string{services.DemoCreatorEmail, services.DemoAssigneeEmail} {
		code, _ := postJSON(t, "/auth/login", map[string]string{"email": email, "password": demoTestPassword})
		assert.Equal(t, http.StatusOK, code, email)
	}
}
