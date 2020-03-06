module massnet.org/mass-wallet

require (
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/btcsuite/go-flags v0.0.0-20150116065318-6c288d648c1c
	github.com/btcsuite/winsvc v1.0.0
	github.com/davecgh/go-spew v1.1.1
	github.com/facebookgo/ensure v0.0.0-20160127193407-b4ab57deab51 // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/facebookgo/subset v0.0.0-20150612182917-8dac2c3c4870 // indirect
	github.com/fastly/go-utils v0.0.0-20180712184237-d95a45783239 // indirect
	github.com/gogo/protobuf v1.2.1
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6
	github.com/golang/protobuf v1.3.2
	github.com/grpc-ecosystem/grpc-gateway v1.9.6
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/lestrrat/go-envload v0.0.0-20180220120943-6ed08b54a570 // indirect
	github.com/lestrrat/go-file-rotatelogs v0.0.0-20180223000712-d3151e2a480f
	github.com/lestrrat/go-strftime v0.0.0-20180220042222-ba3bf9c1d042 // indirect
	github.com/massnetorg/tendermint v1.0.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.2.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.0.0
	github.com/spf13/viper v1.5.0
	github.com/stretchr/testify v1.4.0
	github.com/syndtr/goleveldb v1.0.1-0.20190318030020-c3a204f8e965
	github.com/tebeka/strftime v0.1.3 // indirect
	github.com/tecbot/gorocksdb v0.0.0-20190705090504-162552197222
	golang.org/x/crypto v0.0.0-20191122220453-ac88ee75c92c
	golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7
	google.golang.org/genproto v0.0.0-20190817000702-55e96fffbd48
	google.golang.org/grpc v1.23.0
	gopkg.in/fatih/set.v0 v0.2.1
	gopkg.in/karalabe/cookiejar.v2 v2.0.0-20150724131613-8dcd6a7f4951
)

replace massnet.org/mass-wallet => ./

go 1.13
