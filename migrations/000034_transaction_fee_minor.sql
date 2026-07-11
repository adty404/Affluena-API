-- Optional bank admin fee charged to the SOURCE wallet on a wallet-to-wallet
-- transfer, on top of amount_minor. Lives on the transfer transaction row (no
-- separate expense row, no category). Additive and safe on populated data: the
-- default 0 preserves every existing transfer's balance math (source -amount,
-- dest +amount). Only type='transfer' rows ever carry a non-zero fee_minor;
-- that scope is enforced at the application layer.
ALTER TABLE transactions
	ADD COLUMN fee_minor bigint NOT NULL DEFAULT 0 CHECK (fee_minor >= 0);
