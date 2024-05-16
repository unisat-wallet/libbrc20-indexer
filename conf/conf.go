package conf

import (
	"github.com/btcsuite/btcd/chaincfg"
)

var (
	DEBUG                                    = false
	MODULE_SWAP_SOURCE_INSCRIPTION_ID        = "d2a30f6131324e06b1366876c8c089d7ad2a9c2b0ea971c5b0dc6198615bda2ei0"
	GlobalNetParams                          = &chaincfg.MainNetParams
	TICKS_ENABLED                            = ""
	ENABLE_SELF_MINT_HEIGHT           uint32 = 837090
	ENABLE_SWAP_WITHDRAW_HEIGHT       uint32 = 847090 // fixme: dummy height
)
