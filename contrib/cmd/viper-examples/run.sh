echo "using env variables:"
VPE_A=456 VPE_E_F=qrstuvalphabet go run viper-examples.go

echo "using config file:"
go run viper-examples.go config.json
