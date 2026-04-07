package config

var (
	BASE_HTTP_RPC = []string{
		"https://base-mainnet.g.alchemy.com/v2/ZABfTOT9vAxtJL1gFs5uC", // ALCHEMY
		//	"https://distinguished-blue-violet.base-mainnet.quiknode.pro/48414e814ff45ebdbda8e309f5e23348ebeb43d1/", // QUICKNODE
		"https://base-mainnet.core.chainstack.com/6082f10118001c572f3295fca3d5baed",
		//                              // CHAINSTACK
	}
	BASE_WS_RPC = []string{
		"wss://base-mainnet.g.alchemy.com/v2/ZABfTOT9vAxtJL1gFs5uC",
		//	"wss://distinguished-blue-violet.base-mainnet.quiknode.pro/48414e814ff45ebdbda8e309f5e23348ebeb43d1/",
		"wss://base-mainnet.core.chainstack.com/6082f10118001c572f3295fca3d5baed",
	}

	MAIN_HTTP_RPC = []string{
		"https://mainnet.infura.io/v3/e587127983764e6284261ebf6b4aaedf",
		"https://eth-mainnet.g.alchemy.com/v2/ZABfTOT9vAxtJL1gFs5uC",
	}

	MAIN_WS_RPC = []string{
		"wss://eth-mainnet.g.alchemy.com/v2/ZABfTOT9vAxtJL1gFs5uC",
	}

	INFURAMAIN = "https://mainnet.infura.io/v3/e587127983764e6284261ebf6b4aaedf"
	BASEDRPC   = "https://lb.drpc.live/base/AhuxMhCqfkI8pF_0y4Fpi89GWcIMFIwR8ZsatuZZzRRv"
	PubRPC     = "https://ethereum-rpc.publicnode.com"
)
