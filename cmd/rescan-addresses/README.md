# rescan-addresses

__rescan-address fixes the issue of not being able to create invoices after an BTCPay Server update.__

After updating BTCPay Server, generating an invoice might return an error 500 and leave this in the logs:

```
# docker logs generated_btcpayserver_1
Microsoft.EntityFrameworkCore.DbUpdateException: An error occurred while updating the entries. See the inner exception for details.
 ---> Npgsql.PostgresException (0x80004005): 23505: duplicate key value violates unique constraint "PK_AddressInvoices"
```

BTCPay Server tries to (deterministically) generate a bitcoin address which it had already created in the past.

1. Rescan the wallet (available in the BTCPay user interface). This will move the cursor to the latest bitcoin address which received a payment.
2. Run `rescan-addresses`. It will try to generate invoices until it succeeds, moving the cursor along all bitcoin addresses which had been generated after the latest paid one.

See also [btcpayserver/issues/610](https://github.com/btcpayserver/btcpayserver/issues/610) and [btcpayserver-docker/issues/398](https://github.com/btcpayserver/btcpayserver-docker/issues/398), which suggest to backup NBXplorer data before updating BTCPay Server.
