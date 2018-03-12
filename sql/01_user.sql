DROP TABLE IF EXISTS users;

CREATE TABLE users(
  id  serial,
  email varchar(255) not null default '',
  password  varchar(255) not null default '',
  created_at  timestamp with time zone  not null DEFAULT current_timestamp,
  last_login  timestamp with time zone  not null DEFAULT current_timestamp
);


create unique index email on users(email);
