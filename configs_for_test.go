package command

type RootConfig struct {
	S0 string `flag:"valueName=VAL,required" desc:"String field"`
	B0 bool   `desc:"Bool field"`
}
type Sub1Config struct {
	RootConfig
	S1 string `flag:"required" desc:"String field"`
	B1 bool   `desc:"Bool field"`
}
type Sub2Config struct {
	Sub1Config
	S2 string `desc:"String field"`
	B2 bool   `desc:"Bool field"`
}
type Sub3Config struct {
	Sub2Config
	S3   string   `desc:"String field"`
	B3   bool     `desc:"Bool field"`
	Args []string `flag:"args" desc:"Arbitrary arguments"`
}
