module github.com/heroiclabs/nakama-project-template

go 1.14

// remove google.golang.org/api, just a workarroud for google.golang.org/grpc > 1.30 https://github.com/open-telemetry/opentelemetry-collector/issues/1366#issuecomment-658899511

require (
	cloud.google.com/go/firestore v1.1.1
	firebase.google.com/go v3.13.0+incompatible
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.2 // indirect
	github.com/heroiclabs/nakama-common v1.12.1
	golang.org/x/net v0.0.0-20200925080053-05aa5d4ee321 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	golang.org/x/sys v0.0.0-20200926100807-9d91bd62050c // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/api v0.30.0 // indirect
	google.golang.org/genproto v0.0.0-20201019141844-1ed22bb0c154 // indirect
	google.golang.org/grpc v1.33.1 // indirect
)
