package dkron

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/distribworks/dkron/v2/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobGetParent(t *testing.T) {
	dir, err := ioutil.TempDir("", "dkron-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	a := &Agent{}
	s, err := NewStore(a, dir)
	defer s.Shutdown()
	require.NoError(t, err)
	a.Store = s

	parentTestJob := &Job{
		Name:           "parent_test",
		Executor:       "shell",
		ExecutorConfig: map[string]string{"command": "/bin/false"},
		Schedule:       "@every 2s",
	}

	if err := s.SetJob(parentTestJob, true); err != nil {
		t.Fatalf("error creating job: %s", err)
	}

	dependentTestJob := &Job{
		Name:           "dependent_test",
		Executor:       "shell",
		ExecutorConfig: map[string]string{"command": "/bin/false"},
		ParentJob:      "parent_test",
	}

	err = s.SetJob(dependentTestJob, true)
	assert.NoError(t, err)

	parentTestJob, err = dependentTestJob.GetParent()
	assert.NoError(t, err)
	assert.Equal(t, []string{dependentTestJob.Name}, parentTestJob.DependentJobs)

	ptj, err := dependentTestJob.GetParent()
	assert.NoError(t, err)
	assert.Equal(t, parentTestJob.Name, ptj.Name)

	// Remove the parent job
	dependentTestJob.ParentJob = ""
	dependentTestJob.Schedule = "@every 2m"
	err = s.SetJob(dependentTestJob, true)
	assert.NoError(t, err)

	dtj, _ := s.GetJob(dependentTestJob.Name, nil)
	assert.NoError(t, err)
	assert.Equal(t, "", dtj.ParentJob)

	ptj, err = dtj.GetParent()
	assert.EqualError(t, ErrNoParent, err.Error())

	ptj, err = s.GetJob(parentTestJob.Name, nil)
	assert.NoError(t, err)
	assert.Nil(t, ptj.DependentJobs)
}

func TestNewJobFromProto(t *testing.T) {
	testConfig := map[string]PluginConfig{
		"test_processor": {
			"config_key": "config_value",
		},
	}

	in := &proto.Job{}
	proc := map[string]*proto.PluginConfig{
		"test_processor": {
			Config: map[string]string{"config_key": "config_value"},
		},
	}
	in.Processors = proc

	j := NewJobFromProto(in)
	assert.Equal(t, testConfig, j.Processors)
}

func TestToProto(t *testing.T) {
	j := &Job{
		Processors: map[string]PluginConfig{
			"test_processor": {
				"config_key": "config_value",
			},
		},
	}
	proc := map[string]*proto.PluginConfig{
		"test_processor": {
			Config: map[string]string{"config_key": "config_value"},
		},
	}

	jpb := j.ToProto()
	assert.Equal(t, jpb.Processors, proc)
}

func Test_isRunnable(t *testing.T) {
	dir, err := ioutil.TempDir("", "dkron-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	s, err := NewStore(nil, dir)
	defer s.Shutdown()
	require.NoError(t, err)

	testCases := []struct {
		name string
		job  *Job
		want bool
	}{
		{
			name: "global lock",
			job: &Job{
				Agent: &Agent{
					GlobalLock: true,
					Store:      s,
				},
			},
			want: false,
		},
		{
			name: "running forbid",
			job: &Job{
				Agent: &Agent{
					Store: s,
				},
				Status:      StatusRunning,
				Concurrency: ConcurrencyForbid,
			},
			want: false,
		},
		{
			name: "success forbid",
			job: &Job{
				Agent: &Agent{
					Store: s,
				},
				Status:      StatusRunning,
				Concurrency: ConcurrencyForbid,
			},
			want: false,
		},
		{
			name: "running true",
			job: &Job{
				Agent: &Agent{
					Store: s,
				},
				running: true,
			},
			want: false,
		},
		{
			name: "should run",
			job: &Job{
				Agent: &Agent{
					Store: s,
				},
			},
			want: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.job.isRunnable())
		})
	}
}

func Test_generateJobTree(t *testing.T) {
	jsonString := `[
		{
			"name": "task1",
			"displayname": "",
			"timezone": "",
			"schedule": "0 0 17 * * *",
			"owner": "",
			"owner_email": "",
			"success_count": 26,
			"error_count": 6,
			"last_success": "2019-11-04T04:37:12.396866367Z",
			"last_error": "2019-11-27T15:17:47.925761Z",
			"disabled": false,
			"tags": null,
			"metadata": null,
			"retries": 0,
			"dependent_jobs": null,
			"parent_job": "",
			"processors": {},
			"concurrency": "forbid",
			"executor": "http",
			"executor_config": {
				"body": "{\"partner_id\": \"3123\",\"user_id\": \"14123\"}",
				"debug": "true",
				"expectBody": "",
				"expectCode": "200",
				"headers": "[]",
				"method": "POST",
				"timeout": "30",
				"url": "xxxxxxxxxxxxx"
			},
			"status": "success",
			"next": "2019-11-28T09:00:00Z"
		},
		{
			"name": "task2",
			"displayname": "",
			"timezone": "",
			"schedule": "0 0 17 * * *",
			"owner": "",
			"owner_email": "",
			"success_count": 26,
			"error_count": 6,
			"last_success": "2019-11-04T04:37:12.396866367Z",
			"last_error": "2019-11-27T15:17:47.925761Z",
			"disabled": false,
			"tags": null,
			"metadata": null,
			"retries": 0,
			"dependent_jobs": null,
			"parent_job": "",
			"processors": {},
			"concurrency": "forbid",
			"executor": "http",
			"executor_config": {
				"body": "{\"partner_id\": \"3123\",\"user_id\": \"14123\"}",
				"debug": "true",
				"expectBody": "",
				"expectCode": "200",
				"headers": "[]",
				"method": "POST",
				"timeout": "30",
				"url": "xxxxxxxxxxxxx"
			},
			"status": "success",
			"next": "2019-11-28T09:00:00Z"
		},
		{
			"name": "task3",
			"displayname": "",
			"timezone": "",
			"schedule": "0 0 17 * * *",
			"owner": "",
			"owner_email": "",
			"success_count": 26,
			"error_count": 6,
			"last_success": "2019-11-04T04:37:12.396866367Z",
			"last_error": "2019-11-27T15:17:47.925761Z",
			"disabled": false,
			"tags": null,
			"metadata": null,
			"retries": 0,
			"dependent_jobs": null,
			"parent_job": "task2",
			"processors": {},
			"concurrency": "forbid",
			"executor": "http",
			"executor_config": {
				"body": "{\"partner_id\": \"3123\",\"user_id\": \"14123\"}",
				"debug": "true",
				"expectBody": "",
				"expectCode": "200",
				"headers": "[]",
				"method": "POST",
				"timeout": "30",
				"url": "xxxxxxxxxxxxx"
			},
			"status": "success",
			"next": "2019-11-28T09:00:00Z"
		},
		{
			"name": "task4",
			"displayname": "",
			"timezone": "",
			"schedule": "0 0 17 * * *",
			"owner": "",
			"owner_email": "",
			"success_count": 26,
			"error_count": 6,
			"last_success": "2019-11-04T04:37:12.396866367Z",
			"last_error": "2019-11-27T15:17:47.925761Z",
			"disabled": false,
			"tags": null,
			"metadata": null,
			"retries": 0,
			"dependent_jobs": null,
			"parent_job": "",
			"processors": {},
			"concurrency": "forbid",
			"executor": "http",
			"executor_config": {
				"body": "{\"partner_id\": \"3123\",\"user_id\": \"14123\"}",
				"debug": "true",
				"expectBody": "",
				"expectCode": "200",
				"headers": "[]",
				"method": "POST",
				"timeout": "30",
				"url": "xxxxxxxxxxxxx"
			},
			"status": "success",
			"next": "2019-11-28T09:00:00Z"
		},
		{
			"name": "task5",
			"displayname": "",
			"timezone": "",
			"schedule": "0 0 17 * * *",
			"owner": "",
			"owner_email": "",
			"success_count": 26,
			"error_count": 6,
			"last_success": "2019-11-04T04:37:12.396866367Z",
			"last_error": "2019-11-27T15:17:47.925761Z",
			"disabled": false,
			"tags": null,
			"metadata": null,
			"retries": 0,
			"dependent_jobs": null,
			"parent_job": "task4",
			"processors": {},
			"concurrency": "forbid",
			"executor": "http",
			"executor_config": {
				"body": "{\"partner_id\": \"3123\",\"user_id\": \"14123\"}",
				"debug": "true",
				"expectBody": "",
				"expectCode": "200",
				"headers": "[]",
				"method": "POST",
				"timeout": "30",
				"url": "xxxxxxxxxxxxx"
			},
			"status": "success",
			"next": "2019-11-28T09:00:00Z"
		},
		{
			"name": "task6",
			"displayname": "",
			"timezone": "",
			"schedule": "0 0 17 * * *",
			"owner": "",
			"owner_email": "",
			"success_count": 26,
			"error_count": 6,
			"last_success": "2019-11-04T04:37:12.396866367Z",
			"last_error": "2019-11-27T15:17:47.925761Z",
			"disabled": false,
			"tags": null,
			"metadata": null,
			"retries": 0,
			"dependent_jobs": null,
			"parent_job": "task5",
			"processors": {},
			"concurrency": "forbid",
			"executor": "http",
			"executor_config": {
				"body": "{\"partner_id\": \"3123\",\"user_id\": \"14123\"}",
				"debug": "true",
				"expectBody": "",
				"expectCode": "200",
				"headers": "[]",
				"method": "POST",
				"timeout": "30",
				"url": "xxxxxxxxxxxxx"
			},
			"status": "success",
			"next": "2019-11-28T09:00:00Z"
		},
		{
			"name": "task7",
			"displayname": "",
			"timezone": "",
			"schedule": "0 0 17 * * *",
			"owner": "",
			"owner_email": "",
			"success_count": 26,
			"error_count": 6,
			"last_success": "2019-11-04T04:37:12.396866367Z",
			"last_error": "2019-11-27T15:17:47.925761Z",
			"disabled": false,
			"tags": null,
			"metadata": null,
			"retries": 0,
			"dependent_jobs": null,
			"parent_job": "task5",
			"processors": {},
			"concurrency": "forbid",
			"executor": "http",
			"executor_config": {
				"body": "{\"partner_id\": \"3123\",\"user_id\": \"14123\"}",
				"debug": "true",
				"expectBody": "",
				"expectCode": "200",
				"headers": "[]",
				"method": "POST",
				"timeout": "30",
				"url": "xxxxxxxxxxxxx"
			},
			"status": "success",
			"next": "2019-11-28T09:00:00Z"
		},
		{
			"name": "task8",
			"displayname": "",
			"timezone": "",
			"schedule": "0 0 17 * * *",
			"owner": "",
			"owner_email": "",
			"success_count": 26,
			"error_count": 6,
			"last_success": "2019-11-04T04:37:12.396866367Z",
			"last_error": "2019-11-27T15:17:47.925761Z",
			"disabled": false,
			"tags": null,
			"metadata": null,
			"retries": 0,
			"dependent_jobs": null,
			"parent_job": "task6",
			"processors": {},
			"concurrency": "forbid",
			"executor": "http",
			"executor_config": {
				"body": "{\"partner_id\": \"3123\",\"user_id\": \"14123\"}",
				"debug": "true",
				"expectBody": "",
				"expectCode": "200",
				"headers": "[]",
				"method": "POST",
				"timeout": "30",
				"url": "xxxxxxxxxxxxx"
			},
			"status": "success",
			"next": "2019-11-28T09:00:00Z"
		}
	]`
	var jobs []*Job
	err := json.Unmarshal([]byte(jsonString), &jobs)
	if err != nil {
		t.Fatalf("unmarshal json job error: %s", err)
	}
	jobTree, err := generateJobTree(jobs)
	if err != nil {
		t.Fatalf("generate job tree error: %s", err)
	}
	assert.Equal(t, len(jobTree), 3)

}
