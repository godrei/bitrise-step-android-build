package step

import (
	"testing"
	"time"

	"github.com/bitrise-io/go-android/gradle"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/env"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/bitrise-step-android-build/step/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFilterVariants(t *testing.T) {
	variants := gradle.Variants{
		"module1": []string{"variant1", "variant2", "variant3", "variant4", "variant5", "shared", "shared2"},
		"module2": []string{"2variant1", "2variant2", "shared", "2variant3", "2variant4", "2variant5", "shared2"},
	}

	t.Log("exact match for module and variant")
	{
		filtered, err := filterVariants("module1", "variant3", variants)
		require.NoError(t, err)

		expectedVariants := gradle.Variants{
			"module1": []string{"variant3"},
		}

		require.Equal(t, expectedVariants, filtered)

		_, err = filterVariants("module1", "variant100", variants)
		require.Error(t, err)

		_, err = filterVariants("module100", "variant100", variants)
		require.Error(t, err)

		_, err = filterVariants("module100", "variant1", variants)
		require.Error(t, err)
	}

	t.Log("exact match for module")
	{
		filtered, err := filterVariants("module1", "", variants)
		require.NoError(t, err)

		expectedVariants := gradle.Variants{
			"module1": []string{"variant1", "variant2", "variant3", "variant4", "variant5", "shared", "shared2"},
		}

		require.Equal(t, expectedVariants, filtered)

		_, err = filterVariants("module3", "", variants)
		require.Error(t, err)
	}

	t.Log("exact match for variant")
	{
		filtered, err := filterVariants("", "variant2", variants)
		require.NoError(t, err)

		expectedVariants := gradle.Variants{
			"module1": []string{"variant2"},
		}

		require.Equal(t, expectedVariants, filtered)

		filtered, err = filterVariants("", "", variants)
		require.NoError(t, err)
		require.Equal(t, variants, filtered)

		filtered, err = filterVariants("", "shared", variants)
		require.NoError(t, err)

		expectedVariants = gradle.Variants{
			"module1": []string{"shared"},
			"module2": []string{"shared"},
		}

		require.Equal(t, expectedVariants, filtered)
	}

	t.Log("check no overlapping variants")
	{
		variants := gradle.Variants{
			"module1": []string{"variant1", "variant12"},
		}

		filtered, err := filterVariants("module1", "variant1", variants)
		require.NoError(t, err)

		expectedVariants := gradle.Variants{
			"module1": []string{"variant1"},
		}

		require.Equal(t, expectedVariants, filtered)
	}

	t.Log("exact match for module and multiple variants")
	{
		filtered, err := filterVariants("module1", `variant1\nvariant2`, variants)
		require.NoError(t, err)

		expectedVariants := gradle.Variants{
			"module1": []string{"variant1", "variant2"},
		}

		require.Equal(t, expectedVariants, filtered)
	}

	t.Log("exact match for multiple variants")
	{
		filtered, err := filterVariants("", `shared\nshared2`, variants)
		require.NoError(t, err)

		expectedVariants := gradle.Variants{
			"module1": []string{"shared", "shared2"},
			"module2": []string{"shared", "shared2"},
		}

		require.Equal(t, expectedVariants, filtered)
	}

	t.Log("filter out utility variants")
	{
		variants := gradle.Variants{
			"module1": []string{
				"DemoDebug", "DemoDebugAndroidTestClasses", "DemoDebugAndroidTestResources",
				"DemoDebugClasses", "DemoDebugResources", "DemoDebugUnitTestClasses",
				"DemoRelease", "DemoReleaseClasses", "DemoReleaseResources", "DemoReleaseUnitTestClasses",
			},
		}

		filtered, err := filterVariants("module1", "", variants)
		require.NoError(t, err)

		expectedVariants := gradle.Variants{
			"module1": []string{"DemoDebug", "DemoRelease"},
		}

		require.Equal(t, expectedVariants, filtered)
	}

	t.Log("exact match for module and single not existing variant")
	{
		_, err := filterVariants("module1", "not-existings-variant", variants)
		require.Error(t, err)
	}

	t.Log("single not existing variant")
	{
		_, err := filterVariants("", "not-existings-variant", variants)
		require.Error(t, err)
	}

	t.Log("exact match for module and multiple variants, single not existing")
	{
		_, err := filterVariants("module1", `variant1\nnot-existings-variant`, variants)
		require.Error(t, err)
	}

	t.Log("multiple variants, single not existing")
	{
		_, err := filterVariants("", `variant2\nnot-existings-variant`, variants)
		require.Error(t, err)
	}
}

func TestVariantSeparation(t *testing.T) {
	testCases := []struct {
		title             string
		variantsAsOneLine string
		want              []string
	}{
		{
			"1. Given multiple variants",
			`variant1\nvariant2`,
			[]string{"variant1", "variant2"},
		},
		{
			"2. Given single variant",
			`variant1`,
			[]string{"variant1"},
		},
		{
			"3. Given empty variant",
			``,
			[]string{""},
		},
	}

	for _, testCase := range testCases {
		// When
		variants := separateVariants(testCase.variantsAsOneLine)

		// Then
		require.Equal(t, testCase.want, variants)
	}
}

func Test_GivenMatchingFiles_WhenGettingArtifacts_ThenArtifactsReturned(t *testing.T) {
	// Given
	step := createStep()
	startTime := time.Date(2021, 8, 18, 8, 0, 0, 0, time.UTC)
	appPathPattern := []string{"*/build/outputs/apk/*.apk", "*/build/outputs/bundle/*.aab"}
	gradleWrapper := new(mocks.MockGradleProjectWrapper)
	testArtifacts := []gradle.Artifact{
		{
			Path: "/bitrise/src/app/build/outputs/apk/my-app-debug.apk",
			Name: "my-app-debug.apk",
		},
	}
	gradleWrapper.On("FindArtifacts", startTime, appPathPattern[0], false).Return(testArtifacts, nil)
	gradleWrapper.On("FindArtifacts", startTime, appPathPattern[1], false).Return([]gradle.Artifact{}, nil)

	// When
	artifacts, err := step.getArtifacts(gradleWrapper, startTime, appPathPattern, false)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, testArtifacts, artifacts)
	gradleWrapper.AssertCalled(t, "FindArtifacts", startTime, appPathPattern[0], false)
	gradleWrapper.AssertCalled(t, "FindArtifacts", startTime, appPathPattern[1], false)
}

func Test_GivenNoMatchingFiles_WhenGettingArtifacts_ThenRetryWithoutModTimeCheck(t *testing.T) {
	// Given
	step := createStep()
	startTime := time.Date(2021, 8, 18, 8, 0, 0, 0, time.UTC)
	appPathPattern := []string{"*/build/outputs/apk/*.apk", "*/build/outputs/bundle/*.aab"}
	gradleWrapper := new(mocks.MockGradleProjectWrapper)
	testArtifacts := []gradle.Artifact{
		{
			Path: "/bitrise/src/app/build/outputs/apk/my-app-debug.apk",
			Name: "my-app-debug.apk",
		},
	}
	gradleWrapper.On("FindArtifacts", startTime, mock.Anything, false).Return([]gradle.Artifact{}, nil)
	gradleWrapper.On("FindArtifacts", time.Time{}, appPathPattern[0], false).Return(testArtifacts, nil)
	gradleWrapper.On("FindArtifacts", time.Time{}, appPathPattern[1], false).Return([]gradle.Artifact{}, nil)

	// When
	artifacts, err := step.getArtifacts(gradleWrapper, startTime, appPathPattern, false)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, testArtifacts, artifacts)
	gradleWrapper.AssertCalled(t, "FindArtifacts", startTime, appPathPattern[0], false)
	gradleWrapper.AssertCalled(t, "FindArtifacts", startTime, appPathPattern[1], false)
	gradleWrapper.AssertCalled(t, "FindArtifacts", time.Time{}, appPathPattern[0], false)
	gradleWrapper.AssertCalled(t, "FindArtifacts", time.Time{}, appPathPattern[1], false)
}

func createStep() AndroidBuild {
	envRepository := env.NewRepository()
	return AndroidBuild{
		inputParser: stepconf.NewInputParser(envRepository),
		logger:      log.NewLogger(),
		cmdFactory:  command.NewFactory(envRepository),
	}
}
