-- +goose Up
-- Backfill parent_id on existing Plaid sub-categories based on primary prefix.
-- Parent category rows themselves (cat_food etc.) are seeded by SeedCategories() on startup.
UPDATE categories SET parent_id = 'cat_food'          WHERE parent_id IS NULL AND (id LIKE 'FOOD_AND_DRINK_%' OR id = 'FOOD_AND_DRINK');
UPDATE categories SET parent_id = 'cat_shopping'      WHERE parent_id IS NULL AND (id LIKE 'GENERAL_MERCHANDISE_%' OR id = 'GENERAL_MERCHANDISE');
UPDATE categories SET parent_id = 'cat_transport'     WHERE parent_id IS NULL AND (id LIKE 'TRANSPORTATION_%' OR id = 'TRANSPORTATION');
UPDATE categories SET parent_id = 'cat_health'        WHERE parent_id IS NULL AND (id LIKE 'MEDICAL_%' OR id = 'MEDICAL');
UPDATE categories SET parent_id = 'cat_entertainment' WHERE parent_id IS NULL AND (id LIKE 'ENTERTAINMENT_%' OR id = 'ENTERTAINMENT');
UPDATE categories SET parent_id = 'cat_housing'       WHERE parent_id IS NULL AND (id LIKE 'HOME_IMPROVEMENT_%' OR id = 'HOME_IMPROVEMENT');
UPDATE categories SET parent_id = 'cat_housing'       WHERE parent_id IS NULL AND (id LIKE 'RENT_AND_UTILITIES_%' OR id = 'RENT_AND_UTILITIES');
UPDATE categories SET parent_id = 'cat_personal'      WHERE parent_id IS NULL AND (id LIKE 'PERSONAL_CARE_%' OR id = 'PERSONAL_CARE');
UPDATE categories SET parent_id = 'cat_finance'       WHERE parent_id IS NULL AND (id LIKE 'BANK_FEES_%' OR id = 'BANK_FEES');
UPDATE categories SET parent_id = 'cat_finance'       WHERE parent_id IS NULL AND (id LIKE 'LOAN_PAYMENTS_%' OR id = 'LOAN_PAYMENTS');
UPDATE categories SET parent_id = 'cat_education'     WHERE parent_id IS NULL AND (id LIKE 'EDUCATION_%' OR id = 'EDUCATION');
UPDATE categories SET parent_id = 'cat_travel'        WHERE parent_id IS NULL AND (id LIKE 'TRAVEL_%' OR id = 'TRAVEL');
-- Any remaining sub-category (not a parent, not income/transfer) falls to cat_other
UPDATE categories SET parent_id = 'cat_other'
WHERE parent_id IS NULL
  AND id NOT LIKE 'cat_%'
  AND id NOT LIKE 'INCOME%'
  AND id NOT LIKE 'TRANSFER%';

-- +goose Down
UPDATE categories SET parent_id = NULL WHERE id NOT LIKE 'cat_%';
