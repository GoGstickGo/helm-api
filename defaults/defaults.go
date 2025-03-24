package defaults

var (
	Port       = "8080"
	EnvPrefix  = "test-"
	NameSpace  = "helm-api-pg"
	OutPutDir  = "charts"
	SourceDir  = "source/helm/mariadb"
	HelmDriver = "secrets"
	AwsRegion  = "us-east-1"
	SsmParams  = map[string]string{}
)
