package runner 


type QuoterCache interface {
     QuotePools()
}


type MarketCache interface {
  UpdatePos
  GetSnashot 
  GetMarket
  UpdateOnchainRefresh
  UpdateOraclePrice
  MarketRoutine
  ApiCall
 

}