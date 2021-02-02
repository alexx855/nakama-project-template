module github.com/heroiclabs/nakama-project-template

go 1.14

// remove google.golang.org/api, just a workarroud for google.golang.org/grpc > 1.30
require (
	firebase.google.com/go v3.13.0+incompatible
	firebase.google.com/go/v4 v4.2.0 // indirect
	github.com/golang/protobuf v1.4.3
	github.com/heroiclabs/nakama-common v1.11.0
	golang.org/x/net v0.0.0-20200925080053-05aa5d4ee321 // indirect
	golang.org/x/sys v0.0.0-20200926100807-9d91bd62050c // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/api v0.29.0 // indirect
	google.golang.org/genproto v0.0.0-20201019141844-1ed22bb0c154 // indirect
	google.golang.org/grpc v1.33.1 // indirect
)
