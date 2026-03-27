package app

type preset struct {
	Desc  string
	Args  []string
	Batch []string
}

type globalOptions struct {
	repo string
}

type deployMeta struct {
	Repo          string   `json:"repo"`
	Branch        string   `json:"branch"`
	Commit        string   `json:"commit"`
	CommitFull    string   `json:"commit_full"`
	Dirty         bool     `json:"dirty"`
	Preset        string   `json:"preset,omitempty"`
	Channel       string   `json:"channel,omitempty"`
	DeployedFiles []string `json:"deployed_files,omitempty"`
	DeployedAt    string   `json:"deployed_at"`
}

type gitInfo struct {
	Branch     string
	Commit     string
	CommitFull string
	Dirty      bool
}

var presets = map[string]preset{
	"editor":                      {Desc: "Editor (MSVC, x86_64)", Args: []string{"platform=windows"}},
	"editor-mingw":                {Desc: "Editor (MinGW, x86_64)", Args: []string{"platform=windows", "use_mingw=yes"}},
	"editor-production":           {Desc: "Editor production build (LTO + optimizations)", Args: []string{"platform=windows", "production=yes"}},
	"editor-production-mingw":     {Desc: "Editor production build (MinGW + LTO)", Args: []string{"platform=windows", "use_mingw=yes", "production=yes"}},
	"editor-dev":                  {Desc: "Editor dev build (assertions, no opts, fast iteration)", Args: []string{"platform=windows", "dev_build=yes"}},
	"template-debug":              {Desc: "Export template debug (x86_64)", Args: []string{"platform=windows", "target=template_debug", "arch=x86_64"}},
	"template-release":            {Desc: "Export template release (x86_64)", Args: []string{"platform=windows", "target=template_release", "arch=x86_64"}},
	"template-debug-production":   {Desc: "Export template debug production (x86_64, LTO)", Args: []string{"platform=windows", "target=template_debug", "arch=x86_64", "production=yes"}},
	"template-release-production": {Desc: "Export template release production (x86_64, LTO)", Args: []string{"platform=windows", "target=template_release", "arch=x86_64", "production=yes"}},
	"template-debug-32":           {Desc: "Export template debug (x86_32)", Args: []string{"platform=windows", "target=template_debug", "arch=x86_32"}},
	"template-release-32":         {Desc: "Export template release (x86_32)", Args: []string{"platform=windows", "target=template_release", "arch=x86_32"}},
	"template-debug-arm64":        {Desc: "Export template debug (arm64)", Args: []string{"platform=windows", "target=template_debug", "arch=arm64"}},
	"template-release-arm64":      {Desc: "Export template release (arm64)", Args: []string{"platform=windows", "target=template_release", "arch=arm64"}},
	"templates-all":               {Desc: "All x86_64 export templates (debug + release, production)", Batch: []string{"template-debug-production", "template-release-production"}},
}
