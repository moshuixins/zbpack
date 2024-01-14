package golang

import (
	"bufio"
	"os"
	"path"

	"github.com/moznion/go-optional"
	"github.com/spf13/afero"
	"github.com/zeabur/zbpack/internal/utils"
	"github.com/zeabur/zbpack/pkg/types"
)

type goPlanContext struct {
	Src           afero.Fs
	SubmoduleName string

	GoVersion optional.Option[string]
	Entry     optional.Option[string]

	Serverless optional.Option[bool]
}

func getGoVersion(ctx *goPlanContext) string {
	ver := &ctx.GoVersion
	if goVer, err := ver.Take(); err == nil {
		return goVer
	}

	fs := ctx.Src

	file, err := fs.Open("go.mod")
	if err != nil {
		return ""
	}
	defer func(file afero.File) {
		_ = file.Close()
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 3 && line[:3] == "go " {
			v := line[3:]
			*ver = optional.Some(v)
			return ver.Unwrap()
		}
	}

	*ver = optional.Some("1.18")
	return ver.Unwrap()
}

func getEntry(ctx *goPlanContext) string {
	ent := &ctx.Entry
	if entry, err := ent.Take(); err == nil {
		return entry
	}

	// in a basic go project, we assume the entrypoint is main.go in root directory
	if utils.HasFile(ctx.Src, "main.go") {
		*ent = optional.Some("")
		return ent.Unwrap()
	}

	// if there is no main.go in root directory, we assume it's a monorepo project.
	// in a general monorepo Go repo of service "user-service", the entry point might be `./cmd/user-service/main.go`
	entry := path.Join("cmd", ctx.SubmoduleName, "main.go")
	if utils.HasFile(ctx.Src, entry) {
		*ent = optional.Some(entry)
		return ent.Unwrap()
	}

	// We know it's a Go project, but we don't know how to build it.
	// We'll just return a generic Go plan type.
	*ent = optional.Some("")
	return ""
}

// GetMetaOptions is the options for GetMeta.
type GetMetaOptions struct {
	Src           afero.Fs
	SubmoduleName string
}

func getServerless(ctx *goPlanContext) bool {
	fcEnv := os.Getenv("FORCE_CONTAINERIZED")
	if fcEnv == "true" || fcEnv == "1" {
		return false
	}

	zsEnv := os.Getenv("ZBPACK_SERVERLESS")
	if zsEnv == "true" || zsEnv == "1" {
		return true
	}

	return false
}

// GetMeta gets the metadata of the Go project.
func GetMeta(opt GetMetaOptions) types.PlanMeta {
	ctx := &goPlanContext{Src: opt.Src, SubmoduleName: opt.SubmoduleName}
	meta := types.PlanMeta{}

	goVersion := getGoVersion(ctx)
	meta["goVersion"] = goVersion

	entry := getEntry(ctx)
	meta["entry"] = entry

	serverless := getServerless(ctx)
	if serverless {
		meta["serverless"] = "true"
	}

	return meta
}
