-- name: search-conversations
WITH matched_conversations AS (
    SELECT id
    FROM conversations
    WHERE reference_number = $1

    UNION ALL

    -- $10: LIKE patterns of the search words, longest first. $11: the raw words, same order.
    -- Candidates are found with the longest word so the unaccent trigram indexes are used; the
    -- final WHERE then requires every word to match (any field, any order).
    SELECT id
    FROM conversations
    WHERE f_unaccent(subject) ILIKE f_unaccent(($10::text[])[1]) ESCAPE '\'

    UNION ALL

    SELECT conversations.id
    FROM users
    JOIN conversations ON conversations.contact_id = users.id
    LEFT JOIN companies ON companies.id = users.company_id
    WHERE f_unaccent(users.email) ILIKE f_unaccent(($10::text[])[1]) ESCAPE '\'
       OR f_unaccent(users.first_name) ILIKE f_unaccent(($10::text[])[1]) ESCAPE '\'
       OR f_unaccent(users.last_name) ILIKE f_unaccent(($10::text[])[1]) ESCAPE '\'
       OR f_unaccent(companies.name) ILIKE f_unaccent(($10::text[])[1]) ESCAPE '\'
       OR companies.document ILIKE ($10::text[])[1] ESCAPE '\'
       OR companies.external_id ILIKE ($10::text[])[1] ESCAPE '\'
       OR users.id::text = $1
)
SELECT
    conversations.id,
    conversations.reference_number = $1 AS reference_match,
    conversations.created_at,
    conversations.uuid,
    conversations.reference_number,
    conversations.subject,
    conversations.last_message,
    conversations.last_message_at,
    conversations.assigned_user_id,
    conversations.assigned_team_id,
    assignee.first_name AS "assignee.first_name",
    assignee.last_name AS "assignee.last_name",
    assignee.avatar_url AS "assignee.avatar_url",
    teams.name AS team_name,
    conversation_statuses.name AS status,
    conversation_priorities.name AS priority,
    inboxes.name AS inbox_name,
    inboxes.channel AS inbox_channel,
    users.first_name AS "contact.first_name",
    users.last_name AS "contact.last_name",
    users.email AS "contact.email",
    users.avatar_url AS "contact.avatar_url",
    COALESCE(
        (SELECT ARRAY_AGG(tags.name ORDER BY tags.name) FROM conversation_tags JOIN tags ON tags.id = conversation_tags.tag_id WHERE conversation_tags.conversation_id = conversations.id),
        '{}'
    ) AS tags
FROM conversations
JOIN users ON conversations.contact_id = users.id
LEFT JOIN companies ON companies.id = users.company_id
LEFT JOIN users assignee ON conversations.assigned_user_id = assignee.id
LEFT JOIN teams ON conversations.assigned_team_id = teams.id
LEFT JOIN inboxes ON conversations.inbox_id = inboxes.id
LEFT JOIN conversation_statuses ON conversations.status_id = conversation_statuses.id
LEFT JOIN conversation_priorities ON conversations.priority_id = conversation_priorities.id
WHERE conversations.id IN (SELECT id FROM matched_conversations)
  AND (
       conversations.reference_number = $1
    OR users.id::text = $1
    OR NOT EXISTS (
        SELECT 1 FROM unnest($10::text[]) AS word(pattern)
        WHERE NOT (
               f_unaccent(conversations.subject) ILIKE f_unaccent(word.pattern) ESCAPE '\'
            OR f_unaccent(users.email) ILIKE f_unaccent(word.pattern) ESCAPE '\'
            OR f_unaccent(users.first_name) ILIKE f_unaccent(word.pattern) ESCAPE '\'
            OR f_unaccent(users.last_name) ILIKE f_unaccent(word.pattern) ESCAPE '\'
            OR f_unaccent(companies.name) ILIKE f_unaccent(word.pattern) ESCAPE '\'
            OR companies.document ILIKE word.pattern ESCAPE '\'
            OR companies.external_id ILIKE word.pattern ESCAPE '\'
        )
    )
  )
  AND $11::text[] IS NOT NULL
  AND $3
  AND (
       $4
    OR ($5 AND conversations.assigned_user_id = $2)
    OR ($6 AND conversations.assigned_team_id = ANY($9::int[]))
    OR ($7 AND conversations.assigned_team_id = ANY($9::int[]) AND conversations.assigned_user_id IS NULL)
    OR ($8 AND conversations.assigned_user_id IS NULL AND (conversations.assigned_team_id IS NULL OR conversations.assigned_team_id = ANY($9::int[])))
  )

-- name: search-messages
SELECT
    conversation_messages.id,
    conversation_messages.uuid,
    conversation_messages.created_at,
    conversation_messages.type,
    conversation_messages.private,
    LEFT(conversation_messages.text_content, 200) AS text_content,
    CASE WHEN POSITION(LOWER(f_unaccent(($11::text[])[1])) IN LOWER(f_unaccent(conversation_messages.text_content))) > 81 THEN '…' ELSE '' END
        || SUBSTRING(
            conversation_messages.text_content
            FROM GREATEST(POSITION(LOWER(f_unaccent(($11::text[])[1])) IN LOWER(f_unaccent(conversation_messages.text_content))) - 80, 1)
            FOR 300
        ) AS snippet,
    sender.first_name AS "sender.first_name",
    sender.last_name AS "sender.last_name",
    conversations.created_at AS conversation_created_at,
    conversations.uuid AS conversation_uuid,
    conversations.reference_number AS conversation_reference_number,
    conversations.subject AS conversation_subject,
    conversations.assigned_user_id,
    conversations.assigned_team_id,
    assignee.first_name AS "assignee.first_name",
    assignee.last_name AS "assignee.last_name",
    assignee.avatar_url AS "assignee.avatar_url",
    teams.name AS team_name,
    conversation_statuses.name AS conversation_status,
    conversation_priorities.name AS priority,
    inboxes.name AS inbox_name,
    inboxes.channel AS inbox_channel,
    users.first_name AS "contact.first_name",
    users.last_name AS "contact.last_name",
    users.email AS "contact.email",
    users.avatar_url AS "contact.avatar_url"
FROM conversation_messages
JOIN conversations ON conversation_messages.conversation_id = conversations.id
JOIN users ON conversations.contact_id = users.id
LEFT JOIN users sender ON conversation_messages.sender_id = sender.id
LEFT JOIN users assignee ON conversations.assigned_user_id = assignee.id
LEFT JOIN teams ON conversations.assigned_team_id = teams.id
LEFT JOIN inboxes ON conversations.inbox_id = inboxes.id
LEFT JOIN conversation_statuses ON conversations.status_id = conversation_statuses.id
LEFT JOIN conversation_priorities ON conversations.priority_id = conversation_priorities.id
WHERE conversation_messages.type != 'activity'
  AND $1::text IS NOT NULL
  AND f_unaccent(conversation_messages.text_content) ILIKE f_unaccent(($10::text[])[1]) ESCAPE '\'
  AND NOT EXISTS (
      SELECT 1 FROM unnest($10::text[]) AS word(pattern)
      WHERE f_unaccent(conversation_messages.text_content) NOT ILIKE f_unaccent(word.pattern) ESCAPE '\'
  )
  AND $3
  AND (
       $4
    OR ($5 AND conversations.assigned_user_id = $2)
    OR ($6 AND conversations.assigned_team_id = ANY($9::int[]))
    OR ($7 AND conversations.assigned_team_id = ANY($9::int[]) AND conversations.assigned_user_id IS NULL)
    OR ($8 AND conversations.assigned_user_id IS NULL AND (conversations.assigned_team_id IS NULL OR conversations.assigned_team_id = ANY($9::int[])))
  )

-- name: search-contacts
SELECT
    id,
    created_at,
    first_name,
    last_name,
    email,
    phone_number,
    phone_number_country_code,
    external_user_id
FROM users
WHERE type = 'contact'
AND deleted_at IS NULL
-- $1: LIKE patterns of the search words, longest first. Every word must match some field.
AND (
       f_unaccent(email) ILIKE f_unaccent(($1::text[])[1]) ESCAPE '\'
    OR f_unaccent(phone_number) ILIKE f_unaccent(($1::text[])[1]) ESCAPE '\'
    OR f_unaccent(first_name) ILIKE f_unaccent(($1::text[])[1]) ESCAPE '\'
    OR f_unaccent(last_name) ILIKE f_unaccent(($1::text[])[1]) ESCAPE '\'
    OR EXISTS (
        SELECT 1 FROM companies c
        WHERE c.id = users.company_id
          AND (f_unaccent(c.name) ILIKE f_unaccent(($1::text[])[1]) ESCAPE '\'
            OR c.document ILIKE ($1::text[])[1] ESCAPE '\'
            OR c.external_id ILIKE ($1::text[])[1] ESCAPE '\')
    )
)
AND NOT EXISTS (
    SELECT 1 FROM unnest($1::text[]) AS word(pattern)
    WHERE NOT (
           f_unaccent(email) ILIKE f_unaccent(word.pattern) ESCAPE '\'
        OR f_unaccent(phone_number) ILIKE f_unaccent(word.pattern) ESCAPE '\'
        OR f_unaccent(first_name) ILIKE f_unaccent(word.pattern) ESCAPE '\'
        OR f_unaccent(last_name) ILIKE f_unaccent(word.pattern) ESCAPE '\'
        OR EXISTS (
            SELECT 1 FROM companies c
            WHERE c.id = users.company_id
              AND (f_unaccent(c.name) ILIKE f_unaccent(word.pattern) ESCAPE '\'
                OR c.document ILIKE word.pattern ESCAPE '\'
                OR c.external_id ILIKE word.pattern ESCAPE '\')
        )
    )
)
LIMIT $2;
