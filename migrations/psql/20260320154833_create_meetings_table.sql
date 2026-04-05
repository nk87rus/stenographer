-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS public.meetings
(
    id bigint NOT NULL,
    ts bigint NOT NULL DEFAULT (EXTRACT(epoch FROM now()))::bigint,
    user_id bigint NOT NULL,
    data text,
    CONSTRAINT uniq_id PRIMARY KEY (id),
    CONSTRAINT user_fkey FOREIGN KEY (user_id)
        REFERENCES public.users (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.meetings;
-- +goose StatementEnd
