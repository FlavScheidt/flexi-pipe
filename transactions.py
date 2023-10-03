from xrpl.clients import JsonRpcClient
from xrpl.wallet import generate_faucet_wallet
from xrpl.models.transactions import Payment
from xrpl.utils import xrp_to_drops
from xrpl.wallet import Wallet
from xrpl.constants import CryptoAlgorithm


#Connect
JSON_RPC_URL = "http://localhost:5005/"
client = JsonRpcClient(JSON_RPC_URL)

#Generate wallets
# wallet1 = generate_faucet_wallet(client, debug=True)
# wallet2 = generate_faucet_wallet(client, debug=True)

#Get account numbers
# account1 = wallet1.address
# account2 = wallet2.address

wallet1 = Wallet.from_seed(seed="sn3nxiW7v8KXzPzAqzyHXbSSKNuN9", algorithm=CryptoAlgorithm.SECP256K1)
print(wallet1.address) # "rMCcNuTcajgw7YTgBy1sys3b89QqjUrMpH"

#Prepare payment
my_payment = xrpl.models.transactions.Payment(
    account=account1,
    amount=xrpl.utils.xrp_to_drops(22),
    destination="rPT1Sjq2YGrBMTttX4GZHjKu9dyfzbpAYe",
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