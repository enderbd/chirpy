-- +goose up
alter table users
	add column hashed_password text not null default 'unset';

-- +goose down
alter table users
	drop column hashed_password;
