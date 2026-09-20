# Отношения и функциональные зависимости

ER-схемы: `01_1NF.md`, `02_2NF.md`, `03_3NF_BCNF.md`.

Правила, из которых берутся зависимости:

- у пользователя два ключа: `id` и `login`;
- у заметки один автор и не больше одного родителя;
- блок можно использовать в нескольких заметках, вложение — в нескольких блоках;
- в одной заметке блок не повторяется, в одном блоке вложение не повторяется;
- `position` уникален внутри родителя;
- `creator_login` и `creator_avatar` — текущие данные автора, не снимок;
- `content` — одно значение, внутри него нет отдельных сущностей;
- других нетривиальных зависимостей нет.



## 1НФ

Схема: `01_1NF.md`.

**user** — учётная запись.
**note** — заметка; login и avatar автора пока скопированы в саму заметку.
**favorite** — связь «пользователь избрал заметку».
**note_block_data** — блок внутри заметки: и связь, и данные блока в одной строке.
**block_attachment_data** — вложение внутри блока: и связь, и путь файла в одной строке.

```text
user:
{id} -> login, password_hash, avatar, created_at, updated_at
{login} -> id, password_hash, avatar, created_at, updated_at

note:
{id} -> parent_note_id, created_by, creator_login, creator_avatar, title, header, icon, created_at, updated_at
{created_by} -> creator_login, creator_avatar

favorite:
нет нетривиальных зависимостей

note_block_data:
{note_id, block_id} -> position, content, block_created_at, block_updated_at
{note_id, position} -> block_id, content, block_created_at, block_updated_at
{block_id} -> content, block_created_at, block_updated_at

block_attachment_data:
{block_id, attachment_id} -> position, path
{block_id, position} -> attachment_id, path
{attachment_id} -> path
```

Это 1НФ: в каждой ячейке лежит одно значение, а не список.

Это ещё не 2НФ: данные блока (`content`, даты) стоят в таблице связи и повторяются в каждой заметке, где этот блок используется. Они зависят только от `block_id`, а не от всей пары `{note_id, block_id}`. То же с путём файла: он зависит только от `attachment_id`, а не от `{block_id, attachment_id}`.  

P.S. Для таблицы `block_attachment_data` связи нет, потому что ещё нет уникального родителя для блока. Во 2НФ она появится.

## 2НФ

Схема: `02_2NF.md`.

Данные блока и вложения вынесены в отдельные таблицы. В связях остаются только ключи и порядок.

**block** — содержимое блока.
**attachment** — файл вложения.
**note_block** — вхождение блока в заметку и его `position`.
**block_attachment** — вхождение вложения в блок и его `position`.
`user`, `note`, `favorite` — те же, что в 1НФ.

```text
user:
{id} -> login, password_hash, avatar, created_at, updated_at
{login} -> id, password_hash, avatar, created_at, updated_at

note:
{id} -> parent_note_id, created_by, creator_login, creator_avatar, title, header, icon, created_at, updated_at
{created_by} -> creator_login, creator_avatar

favorite:
нет нетривиальных зависимостей

block:
{id} -> content, created_at, updated_at

note_block:
{note_id, block_id} -> position
{note_id, position} -> block_id

attachment:
{id} -> path

block_attachment:
{block_id, attachment_id} -> position
{block_id, position} -> attachment_id
```

Это 2НФ: неключевой атрибут больше не зависит от части составного ключа. У `note` ключ один — `{id}`, частичных зависимостей у неё и раньше не было.

Это ещё не 3НФ. В `note` есть цепочка `{id} -> created_by` и `{created_by} -> creator_login, creator_avatar`. Автор не ключ заметки: один человек создаёт много заметок. Login и avatar автора — неключевые атрибуты, они зависят от автора, а не от самой заметки.

## 3НФ и НФБК

Схема: `03_3NF_BCNF.md`.

Из `note` убраны `creator_login` и `creator_avatar`. Они живут только в `user`. Отдельная таблица авторов не нужна: это те же `id`, `login`, `avatar`.

**user** — учётная запись; `login` уникален.
**note** — заметка: автор, необязательный родитель, заголовок и оформление.
**favorite** — избранное.
**block** — блок содержимого; может входить в несколько заметок.
**note_block** — какой блок в какой заметке и на каком месте.
**attachment** — вложение.
**block_attachment** — какое вложение в каком блоке и на каком месте.

```text
user:
{id} -> login, password_hash, avatar, created_at, updated_at
{login} -> id, password_hash, avatar, created_at, updated_at

note:
{id} -> parent_note_id, created_by, title, header, icon, created_at, updated_at

favorite:
нет нетривиальных зависимостей

block:
{id} -> content, created_at, updated_at

note_block:
{note_id, block_id} -> position
{note_id, position} -> block_id

attachment:
{id} -> path

block_attachment:
{block_id, attachment_id} -> position
{block_id, position} -> attachment_id
```

Это 3НФ: неключевой атрибут не зависит транзитивно от ключа через другой неключ. В `note` от автора больше ничего не зависит — только ссылка `created_by`.

Это сразу НФБК. НФБК строже 3НФ: левая часть каждой нетривиальной зависимости должна быть ключом. Разница бывает, если часть ключа зависит от того, что само ключом не является. У нас таких зависимостей нет:


| Отношение        | Ключи                                               | Левые части зависимостей       |
| ---------------- | --------------------------------------------------- | ------------------------------ |
| user             | `{id}`, `{login}`                                   | оба ключи                      |
| note             | `{id}`                                              | ключ                           |
| favorite         | `{user_id, note_id}`                                | нетривиальных зависимостей нет |
| block            | `{id}`                                              | ключ                           |
| note_block       | `{note_id, block_id}`, `{note_id, position}`        | оба ключи                      |
| attachment       | `{id}`                                              | ключ                           |
| block_attachment | `{block_id, attachment_id}`, `{block_id, position}` | оба ключи                      |


Поэтому четвёртая схема не нужна: после 3НФ дробить больше нечего.