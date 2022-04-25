module example.com/gowebgo

go 1.18

replace example.com/crypto-client => ./client

replace example.com/crypto-model => ./model

require example.com/crypto-client v0.0.0-00010101000000-000000000000

require example.com/crypto-model v0.0.0-00010101000000-000000000000 // indirect
