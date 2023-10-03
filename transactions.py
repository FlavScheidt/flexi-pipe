import xrpl
from xrpl.clients import JsonRpcClient
from xrpl.wallet import generate_faucet_wallet
from xrpl.models.transactions import Payment
from xrpl.utils import xrp_to_drops
from xrpl.wallet import Wallet
from xrpl.constants import CryptoAlgorithm

import random

#Randomize value
value = random.randint(1, 10)


#Connect
JSON_RPC_URL = "http://localhost:5005/"
client = JsonRpcClient(JSON_RPC_URL)

#Generate wallet
wallet1 = Wallet.from_seed(seed="snoPBrXtMeMyMHUVTgbuqAfg1SUTb", algorithm=CryptoAlgorithm.SECP256K1)
print(wallet1.address) # "rMCcNuTcajgw7YTgBy1sys3b89QqjUrMpH"

#Prepare payment
my_payment = xrpl.models.transactions.Payment(
    account=wallet1.address,
    amount=xrpl.utils.xrp_to_drops(value),
    destination="rnW7CM4K6FcKbW2NcC8j2TQFXn3FAHdwby",
)
print("Payment object:", my_payment)

# Sign transaction
signed_tx = xrpl.transaction.autofill_and_sign(
        my_payment, client, wallet1)

max_ledger = signed_tx.last_ledger_sequence

tx_id = signed_tx.get_hash()

print("Signed transaction:", signed_tx)
print("Transaction cost:", xrpl.utils.drops_to_xrp(signed_tx.fee), "XRP")
print("Transaction expires after ledger:", max_ledger)
print("Identifying hash:", tx_id)

#Submit
try:
    tx_response = xrpl.transaction.submit_and_wait(signed_tx, client)
except xrpl.transaction.XRPLReliableSubmissionException as e:
    exit(f"Submit failed: {e}")