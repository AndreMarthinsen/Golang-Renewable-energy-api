package consts

// External paths

const CountryDomain = "http://129.241.150.113:8080/"
const CountryNamePath = "/v3.1/name/"
const CountryCodePath = "/v3.1/alpha/"
const CountryFilteringByName = "?fields=name,languages,maps,borders&fullText=true"
const CountryFilteringByCode = "?fields=name,languages,maps,borders&codes="

// Internal paths

const DataSetPath = "./internal/assets/renewable-share-energy.csv"

const RenewablesPath = "/energy/" + Version + "/renewables/"
const NotificationPath = "/energy/" + Version + "/notifications/"
const StatusPath = "/energy/" + Version + "/status/"

// Development

const Version = "v1"
const DefaultPort = "10000"
const Development = true
const StubPort = "8888"
const StubDomain = "http://localhost:" + StubPort
