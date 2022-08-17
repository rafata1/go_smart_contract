package main

import (
    "context"
    "crypto/ecdsa"
    "encoding/json"
    "fmt"
    "github.com/ethereum/go-ethereum/common"
    "github.com/gin-gonic/gin"
    "math/big"
    "net/http"
    "strconv"

    "github.com/ethereum/go-ethereum/accounts/abi/bind"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/rafata1/go_smart_contract/api"
)

func getAccountAuth(client *ethclient.Client, accountAddress string) *bind.TransactOpts {

    privateKey, err := crypto.HexToECDSA(accountAddress)
    if err != nil {
        panic(err)
    }

    publicKey := privateKey.Public()
    publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
    if !ok {
        panic("invalid key")
    }

    fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

    //fetch the last use nonce of account
    nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
    if err != nil {
        panic(err)
    }
    fmt.Println("nounce=", nonce)
    chainID, err := client.ChainID(context.Background())
    if err != nil {
        panic(err)
    }

    auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
    if err != nil {
        panic(err)
    }
    auth.Nonce = big.NewInt(int64(nonce))
    auth.Value = big.NewInt(0)      // in wei
    auth.GasLimit = uint64(3000000) // in units
    auth.GasPrice = big.NewInt(1000000)

    return auth
}

func main() {

    client, err := ethclient.Dial("http://127.0.0.1:7545")
    if err != nil {
        panic(err)
    }

    auth := getAccountAuth(client, "f7bfa4bf90cda2597d8b7a84689388cf7ded44f60986ab129283af4c86117174")

    deployedContractAddress, tx, instance, err := api.DeployApi(auth, client)
    if err != nil {
        panic(err)
    }
    fmt.Printf("contract address: %s\n", deployedContractAddress.Hex())
    _, _ = instance, tx
    fmt.Println("instance->", instance)
    fmt.Println("tx->", tx.Hash().Hex())

    conn, err := api.NewApi(common.HexToAddress(deployedContractAddress.Hex()), client)
    r := gin.Default()
    r.GET("/balance", func(c *gin.Context) {
        reply, err := conn.Balance(&bind.CallOpts{})
        if err != nil {
            c.JSON(http.StatusInternalServerError, err.Error())
            return
        }
        c.JSON(http.StatusOK, reply)
        return
    })
    r.GET("/admin", func(c *gin.Context) {
        reply, err := conn.Admin(&bind.CallOpts{})
        if err != nil {
            c.JSON(http.StatusInternalServerError, err.Error())
            return
        }
        c.JSON(http.StatusOK, reply)
        return
    })
    r.POST("/deposit/:amount", func(c *gin.Context) {
        amount, _ := strconv.Atoi(c.Param("amount"))
        var v map[string]interface{}
        err := json.NewDecoder(c.Request.Body).Decode(&v)
        if err != nil {
            c.JSON(http.StatusInternalServerError, err.Error())
            return
        }
        auth := getAccountAuth(client, v["accountPrivateKey"].(string))
        reply, err := conn.Deposite(auth, big.NewInt(int64(amount)))
        if err != nil {
            c.JSON(http.StatusInternalServerError, err.Error())
            return
        }
        c.JSON(http.StatusOK, reply)
        return
    })
    r.POST("/withdrawal/:amount", func(c *gin.Context) {
        amount, _ := strconv.Atoi(c.Param("amount"))
        var v map[string]interface{}
        err := json.NewDecoder(c.Request.Body).Decode(&v)
        if err != nil {
            c.JSON(http.StatusInternalServerError, err.Error())
            return
        }
        auth := getAccountAuth(client, v["accountPrivateKey"].(string))
        reply, err := conn.Withdrawl(auth, big.NewInt(int64(amount)))
        if err != nil {
            c.JSON(http.StatusInternalServerError, err.Error())
            return
        }
        c.JSON(http.StatusOK, reply)
        return
    })
    err = r.Run()
    if err != nil {
        panic(err)
        return
    }
}
