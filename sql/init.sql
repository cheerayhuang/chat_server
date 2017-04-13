CREATE TABLE IF NOT EXISTS chat_users(
    id bigint NOT NULL AUTO_INCREMENT,
    user_name varchar(128) NOT NULL,
    passwd varchar(32) NOT NULL,
    user_type int NOT NULL,
    PRIMARY KEY(id)
)ENGINE = innoDB DEFAULT CHARACTER SET = utf8;
