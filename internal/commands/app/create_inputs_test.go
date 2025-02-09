package app

import (
	"errors"
	"os"
	"path"
	"testing"

	"github.com/10gen/realm-cli/internal/cloud/atlas"
	"github.com/10gen/realm-cli/internal/cloud/realm"
	"github.com/10gen/realm-cli/internal/local"
	"github.com/10gen/realm-cli/internal/utils/test/assert"
	"github.com/10gen/realm-cli/internal/utils/test/mock"

	"github.com/Netflix/go-expect"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAppCreateInputsResolve(t *testing.T) {
	t.Run("with no flags set should prompt for just name and set location deployment model and environment to defaults", func(t *testing.T) {
		profile := mock.NewProfile(t)

		_, console, _, ui, consoleErr := mock.NewVT10XConsole()
		assert.Nil(t, consoleErr)
		defer console.Close()

		procedure := func(c *expect.Console) {
			c.ExpectString("App Name")
			c.SendLine("test-app")
			c.ExpectEOF()
		}

		doneCh := make(chan (struct{}))
		go func() {
			defer close(doneCh)
			procedure(console)
		}()

		inputs := createInputs{}
		assert.Nil(t, inputs.Resolve(profile, ui))

		console.Tty().Close() // flush the writers
		<-doneCh              // wait for procedure to complete

		assert.Equal(t, "test-app", inputs.Name)
		assert.Equal(t, flagDeploymentModelDefault, inputs.DeploymentModel)
		assert.Equal(t, flagLocationDefault, inputs.Location)
		assert.Equal(t, realm.EnvironmentNone, inputs.Environment)
	})
	t.Run("with a name flag set should prompt for nothing else and set location deployment model and environment to defaults", func(t *testing.T) {
		profile := mock.NewProfile(t)

		inputs := createInputs{newAppInputs: newAppInputs{Name: "test-app"}}
		assert.Nil(t, inputs.Resolve(profile, nil))

		assert.Equal(t, "test-app", inputs.Name)
		assert.Equal(t, flagDeploymentModelDefault, inputs.DeploymentModel)
		assert.Equal(t, flagLocationDefault, inputs.Location)
		assert.Equal(t, realm.EnvironmentNone, inputs.Environment)
	})
	t.Run("with name location deployment model and environment flags set should prompt for nothing else", func(t *testing.T) {
		profile := mock.NewProfile(t)

		inputs := createInputs{newAppInputs: newAppInputs{
			Name:            "test-app",
			DeploymentModel: realm.DeploymentModelLocal,
			Location:        realm.LocationOregon,
			Environment:     realm.EnvironmentDevelopment,
		}}
		assert.Nil(t, inputs.Resolve(profile, nil))

		assert.Equal(t, "test-app", inputs.Name)
		assert.Equal(t, realm.DeploymentModelLocal, inputs.DeploymentModel)
		assert.Equal(t, realm.LocationOregon, inputs.Location)
		assert.Equal(t, realm.EnvironmentDevelopment, inputs.Environment)
	})
}

func TestAppCreateInputsResolveName(t *testing.T) {
	testApp := realm.App{
		ID:          primitive.NewObjectID().Hex(),
		GroupID:     primitive.NewObjectID().Hex(),
		ClientAppID: "test-app-abcde",
		Name:        "test-app",
	}

	for _, tc := range []struct {
		description    string
		inputs         createInputs
		appRemote      appRemote
		expectedName   string
		expectedFilter realm.AppFilter
	}{
		{
			description:  "should return name if name is set",
			inputs:       createInputs{newAppInputs: newAppInputs{Name: testApp.Name}},
			expectedName: testApp.Name,
		},
		{
			description:    "should use remote app for name if name is not set",
			appRemote:      appRemote{testApp.GroupID, testApp.ID},
			expectedName:   testApp.Name,
			expectedFilter: realm.AppFilter{GroupID: testApp.GroupID, App: testApp.ID},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			var appFilter realm.AppFilter
			rc := mock.RealmClient{}
			rc.FindAppsFn = func(filter realm.AppFilter) ([]realm.App, error) {
				appFilter = filter
				return []realm.App{testApp}, nil
			}

			err := tc.inputs.resolveName(nil, rc, tc.appRemote)

			assert.Nil(t, err)
			assert.Equal(t, tc.expectedName, tc.inputs.Name)
			assert.Equal(t, tc.expectedFilter, appFilter)
		})
	}

	t.Run("should error when finding app", func(t *testing.T) {
		var appFilter realm.AppFilter
		rc := mock.RealmClient{}
		rc.FindAppsFn = func(filter realm.AppFilter) ([]realm.App, error) {
			appFilter = filter
			return nil, errors.New("realm client error")
		}
		inputs := createInputs{}
		err := inputs.resolveName(nil, rc, appRemote{testApp.GroupID, testApp.ID})

		assert.Equal(t, errors.New("realm client error"), err)
		assert.Equal(t, "", inputs.Name)
		assert.Equal(t, realm.AppFilter{GroupID: testApp.GroupID, App: testApp.ID}, appFilter)
	})
}

func TestAppCreateInputsResolveDirectory(t *testing.T) {
	t.Run("should return path of wd with app name appended", func(t *testing.T) {
		profile := mock.NewProfileFromWd(t)

		appName := "test-app"
		inputs := createInputs{newAppInputs: newAppInputs{Name: appName}}

		dir, err := inputs.resolveLocalPath(nil, profile.WorkingDirectory)

		assert.Nil(t, err)
		assert.Equal(t, path.Join(profile.WorkingDirectory, appName), dir)
	})

	t.Run("should return path of wd with directory appended when local path is set", func(t *testing.T) {
		profile := mock.NewProfileFromWd(t)

		specifiedPath := "test-dir"
		inputs := createInputs{LocalPath: specifiedPath}

		dir, err := inputs.resolveLocalPath(nil, profile.WorkingDirectory)

		assert.Nil(t, err)
		assert.Equal(t, path.Join(profile.WorkingDirectory, specifiedPath), dir)
	})

	t.Run("should return path of wd with app name appended even with file of app name in wd", func(t *testing.T) {
		profile, teardown := mock.NewProfileFromTmpDir(t, "app_create_test")
		defer teardown()

		appName := "test-app"
		inputs := createInputs{newAppInputs: newAppInputs{Name: appName}}

		testFile, err := os.Create(appName)
		assert.Nil(t, err)
		assert.Nil(t, testFile.Close())

		dir, err := inputs.resolveLocalPath(nil, profile.WorkingDirectory)

		assert.Nil(t, err)
		assert.Equal(t, path.Join(profile.WorkingDirectory, appName), dir)
		assert.Nil(t, os.Remove(appName))
	})

	t.Run("should return path of wd with a new app name appended trying to write to a local directory", func(t *testing.T) {
		profile, teardown := mock.NewProfileFromTmpDir(t, "app_create_test")
		defer teardown()

		_, console, _, ui, err := mock.NewVT10XConsole()
		assert.Nil(t, err)
		defer console.Close()

		doneCh := make(chan (struct{}))
		go func() {
			defer close(doneCh)

			console.ExpectString("Local path './test-app' already exists, writing app contents to that destination may result in file conflicts.")
			console.ExpectString("Would you still like to write app contents to './test-app'? ('No' will prompt you to provide another destination)")
			console.SendLine("no")
			console.ExpectString("Local Path")
			console.SendLine("new-app")
			console.ExpectEOF()
		}()

		inputs := createInputs{newAppInputs: newAppInputs{Name: "test-app"}}

		err = os.Mkdir(path.Join(profile.WorkingDirectory, "test-app"), os.ModePerm)
		assert.Nil(t, err)

		dir, err := inputs.resolveLocalPath(ui, profile.WorkingDirectory)
		assert.Nil(t, err)
		assert.Equal(t, path.Join(profile.WorkingDirectory, "new-app"), dir)
		assert.Equal(t, "new-app", inputs.LocalPath)
	})

	t.Run("should error when path specified is another realm app", func(t *testing.T) {
		profile, teardown := mock.NewProfileFromTmpDir(t, "app_create_test")
		defer teardown()

		specifiedDir := "test-dir"
		inputs := createInputs{LocalPath: specifiedDir}
		fullDir := path.Join(profile.WorkingDirectory, specifiedDir)

		appLocal := local.NewApp(
			fullDir,
			"test-app-abcde",
			"test-app",
			flagLocationDefault,
			flagDeploymentModelDefault,
			realm.EnvironmentNone,
			realm.DefaultAppConfigVersion,
		)
		assert.Nil(t, appLocal.WriteConfig())

		dir, err := inputs.resolveLocalPath(nil, profile.WorkingDirectory)

		assert.Equal(t, "", dir)
		assert.Equal(t, errProjectExists{fullDir}, err)
	})
}

func TestAppCreateInputsResolveCluster(t *testing.T) {
	t.Run("should return data source config of a provided cluster", func(t *testing.T) {
		var expectedGroupID string
		ac := mock.AtlasClient{}
		ac.ClustersFn = func(groupID string) ([]atlas.Cluster, error) {
			expectedGroupID = groupID
			return []atlas.Cluster{{ID: "789", Name: "test-cluster"}}, nil
		}

		inputs := createInputs{newAppInputs: newAppInputs{Name: "test-app"}, Cluster: "test-cluster"}

		ds, err := inputs.resolveCluster(ac, "123")
		assert.Nil(t, err)

		assert.Equal(t, dataSourceCluster{
			Name: "mongodb-atlas",
			Type: "mongodb-atlas",
			Config: configCluster{
				ClusterName:         "test-cluster",
				ReadPreference:      "primary",
				WireProtocolEnabled: false,
			},
		}, ds)
		assert.Equal(t, "123", expectedGroupID)
	})

	t.Run("should not be able to find specified cluster", func(t *testing.T) {
		var expectedGroupID string
		ac := mock.AtlasClient{}
		ac.ClustersFn = func(groupID string) ([]atlas.Cluster, error) {
			expectedGroupID = groupID
			return nil, nil
		}

		inputs := createInputs{Cluster: "test-cluster"}

		_, err := inputs.resolveCluster(ac, "123")
		assert.Equal(t, errors.New("failed to find Atlas cluster"), err)
		assert.Equal(t, "123", expectedGroupID)
	})

	t.Run("should error from client", func(t *testing.T) {
		var expectedGroupID string
		ac := mock.AtlasClient{}
		ac.ClustersFn = func(groupID string) ([]atlas.Cluster, error) {
			expectedGroupID = groupID
			return nil, errors.New("client error")
		}

		inputs := createInputs{Cluster: "test-cluster"}

		_, err := inputs.resolveCluster(ac, "123")
		assert.Equal(t, errors.New("client error"), err)
		assert.Equal(t, "123", expectedGroupID)
	})
}

func TestAppCreateInputsResolveDataLake(t *testing.T) {
	t.Run("should return data source config of a provided data lake", func(t *testing.T) {
		var expectedGroupID string
		ac := mock.AtlasClient{}
		ac.DataLakesFn = func(groupID string) ([]atlas.DataLake, error) {
			expectedGroupID = groupID
			return []atlas.DataLake{{Name: "test-datalake"}}, nil
		}

		inputs := createInputs{newAppInputs: newAppInputs{Name: "test-app"}, DataLake: "test-datalake"}

		ds, err := inputs.resolveDataLake(ac, "123")
		assert.Nil(t, err)

		assert.Equal(t, dataSourceDataLake{
			Name: "mongodb-datalake",
			Type: "datalake",
			Config: configDataLake{
				DataLakeName: "test-datalake",
			},
		}, ds)
		assert.Equal(t, "123", expectedGroupID)
	})

	t.Run("should not be able to find specified data lake", func(t *testing.T) {
		var expectedGroupID string
		ac := mock.AtlasClient{}
		ac.DataLakesFn = func(groupID string) ([]atlas.DataLake, error) {
			expectedGroupID = groupID
			return nil, nil
		}

		inputs := createInputs{DataLake: "test-datalake"}

		_, err := inputs.resolveDataLake(ac, "123")
		assert.Equal(t, errors.New("failed to find Atlas data lake"), err)
		assert.Equal(t, "123", expectedGroupID)
	})

	t.Run("should error from client", func(t *testing.T) {
		var expectedGroupID string
		ac := mock.AtlasClient{}
		ac.DataLakesFn = func(groupID string) ([]atlas.DataLake, error) {
			expectedGroupID = groupID
			return nil, errors.New("client error")
		}

		inputs := createInputs{DataLake: "test-datalake"}

		_, err := inputs.resolveDataLake(ac, "123")
		assert.Equal(t, errors.New("client error"), err)
		assert.Equal(t, "123", expectedGroupID)
	})
}
