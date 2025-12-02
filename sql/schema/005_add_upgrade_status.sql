-- +goose up
alter table users
	add column is_chirpy_red BOOLEAN not null default false;

-- +goose down
alter table users
	drop column is_chirpy_red;
