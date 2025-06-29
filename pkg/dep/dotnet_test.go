package dep

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDotnetManager(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create a test .csproj file
	csprojContent := `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net6.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Microsoft.AspNetCore.App" />
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
    <PackageReference Include="Serilog" Version="2.12.0" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(tmpDir+"/test.csproj", []byte(csprojContent), 0600)
	require.NoError(t, err)

	// Create a test packages.lock.json file
	lockContent := `{
  "version": 1,
  "targets": {
    "net6.0": {
      "Newtonsoft.Json/13.0.1": {
        "type": "package",
        "dependencies": {}
      },
      "Serilog/2.12.0": {
        "type": "package",
        "dependencies": {
          "Serilog": "2.12.0"
        }
      }
    }
  }
}`

	err = os.WriteFile(tmpDir+"/packages.lock.json", []byte(lockContent), 0600)
	require.NoError(t, err)

	// Create the manager
	fsys := os.DirFS(tmpDir)
	manager := newDotnetManager(fsys, ".")

	// Initialize the manager
	err = manager.Init()
	require.NoError(t, err)

	// Test Get method
	t.Run("Get", func(t *testing.T) {
		dep, found := manager.Get("Newtonsoft.Json")
		assert.True(t, found)
		assert.Equal(t, "Newtonsoft.Json", dep.Name)
		assert.Equal(t, "13.0.1", dep.Constraint)
		assert.Equal(t, "13.0.1", dep.Version)

		dep, found = manager.Get("Serilog")
		assert.True(t, found)
		assert.Equal(t, "Serilog", dep.Name)
		assert.Equal(t, "2.12.0", dep.Constraint)
		assert.Equal(t, "2.12.0", dep.Version)

		// Test non-existent package
		_, found = manager.Get("NonExistentPackage")
		assert.False(t, found)
	})

	// Test Find method
	t.Run("Find", func(t *testing.T) {
		deps := manager.Find("*")
		assert.Len(t, deps, 3) // Microsoft.AspNetCore.App, Newtonsoft.Json, Serilog

		deps = manager.Find("Newtonsoft.*")
		assert.Len(t, deps, 1)
		assert.Equal(t, "Newtonsoft.Json", deps[0].Name)

		deps = manager.Find("Serilog")
		assert.Len(t, deps, 1)
		assert.Equal(t, "Serilog", deps[0].Name)
	})
}

func TestDotnetManagerWithoutLockFile(t *testing.T) {
	// Create a temporary directory with only .csproj file
	tmpDir := t.TempDir()

	// Create a test .csproj file
	csprojContent := `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net6.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(tmpDir+"/test.csproj", []byte(csprojContent), 0600)
	require.NoError(t, err)

	// Create the manager
	fsys := os.DirFS(tmpDir)
	manager := newDotnetManager(fsys, ".")

	// Initialize the manager
	err = manager.Init()
	require.NoError(t, err)

	// Test Get method without lock file
	dep, found := manager.Get("Newtonsoft.Json")
	assert.True(t, found)
	assert.Equal(t, "Newtonsoft.Json", dep.Name)
	assert.Equal(t, "13.0.1", dep.Constraint)
	assert.Equal(t, "", dep.Version) // No lock file, so no resolved version
}

func TestDotnetManagerRegistration(t *testing.T) {
	// Test that the .NET manager is properly registered
	manager, err := GetManager(ManagerTypeDotnet, os.DirFS("."), ".")
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Test that it's the correct type
	_, ok := manager.(*dotnetManager)
	assert.True(t, ok)
}

func TestDotnetManagerRealisticExample(t *testing.T) {
	// Create a temporary directory with a realistic .NET project structure
	tmpDir := t.TempDir()

	// Create a realistic .csproj file
	csprojContent := `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <Nullable>enable</Nullable>
    <ImplicitUsings>enable</ImplicitUsings>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Microsoft.AspNetCore.OpenApi" Version="8.0.0" />
    <PackageReference Include="Swashbuckle.AspNetCore" Version="6.5.0" />
    <PackageReference Include="Microsoft.EntityFrameworkCore" Version="8.0.0" />
    <PackageReference Include="Microsoft.EntityFrameworkCore.SqlServer" Version="8.0.0" />
    <PackageReference Include="Serilog.AspNetCore" Version="8.0.0" />
    <PackageReference Include="Serilog.Sinks.Console" Version="5.0.0" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(tmpDir+"/MyApp.csproj", []byte(csprojContent), 0600)
	require.NoError(t, err)

	// Create a realistic packages.lock.json file
	lockContent := `{
  "version": 1,
  "targets": {
    "net8.0": {
      "Microsoft.AspNetCore.OpenApi/8.0.0": {
        "type": "package",
        "dependencies": {
          "Microsoft.AspNetCore.Http.Abstractions": "2.2.0"
        }
      },
      "Swashbuckle.AspNetCore/6.5.0": {
        "type": "package",
        "dependencies": {
          "Microsoft.AspNetCore.Mvc.Core": "2.2.5"
        }
      },
      "Microsoft.EntityFrameworkCore/8.0.0": {
        "type": "package",
        "dependencies": {
          "Microsoft.EntityFrameworkCore.Abstractions": "8.0.0"
        }
      },
      "Microsoft.EntityFrameworkCore.SqlServer/8.0.0": {
        "type": "package",
        "dependencies": {
          "Microsoft.EntityFrameworkCore": "8.0.0"
        }
      },
      "Serilog.AspNetCore/8.0.0": {
        "type": "package",
        "dependencies": {
          "Serilog": "3.1.1"
        }
      },
      "Serilog.Sinks.Console/5.0.0": {
        "type": "package",
        "dependencies": {
          "Serilog": "3.0.1"
        }
      }
    }
  }
}`

	err = os.WriteFile(tmpDir+"/packages.lock.json", []byte(lockContent), 0600)
	require.NoError(t, err)

	// Create the manager
	fsys := os.DirFS(tmpDir)
	manager := newDotnetManager(fsys, ".")

	// Initialize the manager
	err = manager.Init()
	require.NoError(t, err)

	// Test finding all dependencies
	deps := manager.Find("*")
	assert.Len(t, deps, 6)

	// Test specific package lookups
	dep, found := manager.Get("Microsoft.AspNetCore.OpenApi")
	assert.True(t, found)
	assert.Equal(t, "Microsoft.AspNetCore.OpenApi", dep.Name)
	assert.Equal(t, "8.0.0", dep.Constraint)
	assert.Equal(t, "8.0.0", dep.Version)

	dep, found = manager.Get("Serilog.AspNetCore")
	assert.True(t, found)
	assert.Equal(t, "Serilog.AspNetCore", dep.Name)
	assert.Equal(t, "8.0.0", dep.Constraint)
	assert.Equal(t, "8.0.0", dep.Version)

	// Test wildcard patterns
	efCoreDeps := manager.Find("Microsoft.EntityFrameworkCore*")
	assert.Len(t, efCoreDeps, 2)

	serilogDeps := manager.Find("Serilog*")
	assert.Len(t, serilogDeps, 2)

	// Test non-existent package
	_, found = manager.Get("NonExistentPackage")
	assert.False(t, found)
}
