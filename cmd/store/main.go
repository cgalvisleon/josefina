package main

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/jsql"
	_ "github.com/cgalvisleon/et/jsql/drivers/oracle" // side-effect: registers the oracle driver
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/josefina/internal/store"
)

func main() {
	dbOss, err := jsql.ConnectTo(jsql.ConnectParams{
		Driver: jsql.DriverOracle,
		Host:   envar.GetStr("ORA_DB_HOST", ""),
		Name:   "ossdb",
		Connection: &jsql.OracleConection{
			Host:        envar.GetStr("ORA_DB_HOST", ""),
			Port:        envar.GetInt("ORA_DB_PORT", 1521),
			Username:    envar.GetStr("ORA_DB_USER", ""),
			Password:    envar.GetStr("ORA_DB_PASSWORD", ""),
			ServiceName: envar.GetStr("ORA_DB_SERVICE_NAME_ORACLE", ""),
			SSL:         envar.GetBool("ORA_DB_SSL_ORACLE", false),
			SSLVerify:   envar.GetBool("ORA_DB_SSL_VERIFY_ORACLE", false),
		},
	})
	if err != nil {
		logs.Panic(err)
	}

	pathData := "./data/collections"
	pathWal := "./data/wal"
	testDb, err := store.Open(pathData, pathWal, "test", store.ReadWrite)
	if err != nil {
		logs.Panic(err)
	}

	for i := 0; i < 100; i++ {
		offset := 1000
		sql := fmt.Sprintf(`
	SELECT 
	SC.ID "id",
	SC.NOMBRE "name", 
	SC.STATUSID "statusid",
	SC.CIUDAD "city", 
	SC.BARRIO "neighborhood", 
	SC.DIRECCION "address", 
	SC.TIPO_DE_DOCUMENTO "document_type", 
	SC.TELEFONO_MOVIL "mobile_phone", 
	SC.CORREO_ELECTRONICO "email", 
	SC.DEPARTAMENTO "department", 
	SC.CREATIONDATE "creationdate", 
	SC.MODIFICATIONDATE "modificationdate", 
	SC.ID_CUENTA "id_cuenta", 
	SC.FRAME "frame",
	SC.SLOT "slot", 
	SC.ONT_ID "ont_id", 
	SC.PRX_INSTALACION_ONT "prx_installation_ont", 
	SC.CANAL_VENTA "sales_channel", 
	SC.ESTRATO "stratum", 
	SC.NÚMERO_DE_DOCUMENTO "document_number", 
	SC.IP "ip", 
	SC.SERIAL_CELSIA "serial_celsia", 
	SC.SERIAL_ONT "serial_ont",
	SC.PUERTO_OLT "olt_port", 
	SC.AMPLIFICADOR "amplifier",
	SC.ENVIO_DE_FACTURACION "billing_shipping",
	SC.PLAN_COMERCIAL "plan_commercial",
	SC.SCCUSTOMERTYPEID "customer_type_id"
	FROM SCCUSTOMER SC	
	ORDER BY SC.ID
	OFFSET %d ROWS
	FETCH FIRST %d ROWS ONLY`, i*offset, offset)
		items, err := dbOss.Sql(sql)
		if err != nil {
			logs.Panic(err)
		}

		now := timezone.Now()
		for _, item := range items.Result {
			id := item.Str("id")
			_, _, err = testDb.Put(id, item)
			if err != nil {
				panic(err)
			}
		}

		elapsed := time.Since(now)
		logs.Debugf("time elapsed: %s for %d items", elapsed.String(), offset)
	}

	// test.ForEach(func(idx string, data []byte) (bool, error) {
	// 	var item et.Json
	// 	err := json.Unmarshal(data, &item)
	// 	if err != nil {
	// 		return false, err
	// 	}

	// 	logs.Debug(item.ToString())
	// 	return true, nil
	// }, true, 0, 0)
}
