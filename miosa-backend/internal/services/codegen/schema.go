// internal/services/codegen/schema.go
//
// Concrete implementation of the codegen contracts.
// Pulls schema definitions from a Groq API endpoint and maps them into
// strongly-typed Go structs used by the generators.
//
// Designed for clarity, immutability of exposed slices, and graceful error handling.

package codegen

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
)

// -----------------------------------------------------------------------------
// Concrete implementations for codegen.Schema, codegen.Entity, codegen.Field, codegen.Relation
// -----------------------------------------------------------------------------

type schemaImpl struct {
    version  string
    entities []entityImpl
}

func (s *schemaImpl) Version() string              { return s.version }
func (s *schemaImpl) Entities() []Entity {
    out := make([]Entity, len(s.entities))
    for i := range s.entities {
        out[i] = &s.entities[i]
    }
    return out
}
func (s *schemaImpl) FindEntity(name string) (Entity, bool) {
    for i := range s.entities {
        if strings.EqualFold(s.entities[i].name, name) {
            return &s.entities[i], true
        }
    }
    return nil, false
}

type entityImpl struct {
    name      string
    table     string
    fields    []fieldImpl
    relations []relationImpl
    doc       string
}

func (e *entityImpl) Name() string                 { return e.name }
func (e *entityImpl) Table() string                { return e.table }
func (e *entityImpl) Fields() []Field {
    out := make([]Field, len(e.fields))
    for i := range e.fields {
        out[i] = &e.fields[i]
    }
    return out
}
func (e *entityImpl) Relations() []Relation {
    out := make([]Relation, len(e.relations))
    for i := range e.relations {
        out[i] = &e.relations[i]
    }
    return out
}
func (e *entityImpl) Doc() string                  { return e.doc }

type fieldImpl struct {
    name         string
    column       string
    kind         string
    format       string
    nullable     bool
    primaryKey   bool
    unique       bool
    defaultValue *string
    maxLen       *int
    scale        *int
    precision    *int
    tags         map[string]string
    doc          string
}

func (f *fieldImpl) Name() string                        { return f.name }
func (f *fieldImpl) Column() string                      { return f.column }
func (f *fieldImpl) Kind() string                        { return f.kind }
func (f *fieldImpl) Format() string                      { return f.format }
func (f *fieldImpl) Nullable() bool                      { return f.nullable }
func (f *fieldImpl) PrimaryKey() bool                    { return f.primaryKey }
func (f *fieldImpl) Unique() bool                        { return f.unique }
func (f *fieldImpl) Default() (string, bool)             { if f.defaultValue != nil { return *f.defaultValue, true }; return "", false }
func (f *fieldImpl) MaxLen() (int, bool)                  { if f.maxLen != nil { return *f.maxLen, true }; return 0, false }
func (f *fieldImpl) ScalePrecision() (int, int, bool)     { if f.scale != nil && f.precision != nil { return *f.scale, *f.precision, true }; return 0, 0, false }
func (f *fieldImpl) Tags() map[string]string              { return f.tags }
func (f *fieldImpl) Doc() string                         { return f.doc }

type relationImpl struct {
    rtype   string
    fromEnt string
    fromFld string
    toEnt   string
    toFld   string
    through *string
    doc     string
}

func (r *relationImpl) Type() string               { return r.rtype }
func (r *relationImpl) From() (string, string)     { return r.fromEnt, r.fromFld }
func (r *relationImpl) To() (string, string)       { return r.toEnt, r.toFld }
func (r *relationImpl) Through() (string, bool)    { if r.through != nil { return *r.through, true }; return "", false }
func (r *relationImpl) Doc() string                { return r.doc }

// -----------------------------------------------------------------------------
// Groq API JSON contract
// -----------------------------------------------------------------------------

type groqSchemaResponse struct {
    Version  string           `json:"version"`
    Entities []groqEntityJSON `json:"entities"`
}

type groqEntityJSON struct {
    Name      string           `json:"name"`
    Table     string           `json:"table"`
    Doc       string           `json:"doc"`
    Fields    []groqFieldJSON  `json:"fields"`
    Relations []groqRelJSON    `json:"relations"`
}

type groqFieldJSON struct {
    Name       string            `json:"name"`
    Column     string            `json:"column"`
    Kind       string            `json:"kind"`
    Format     string            `json:"format"`
    Nullable   bool              `json:"nullable"`
    PrimaryKey bool              `json:"primaryKey"`
    Unique     bool              `json:"unique"`
    Default    *string           `json:"default"`
    MaxLen     *int              `json:"maxLen"`
    Scale      *int              `json:"scale"`
    Precision  *int              `json:"precision"`
    Tags       map[string]string `json:"tags"`
    Doc        string            `json:"doc"`
}

type groqRelJSON struct {
    Type    string  `json:"type"`
    FromEnt string  `json:"fromEntity"`
    FromFld string  `json:"fromField"`
    ToEnt   string  `json:"toEntity"`
    ToFld   string  `json:"toField"`
    Through *string `json:"through"`
    Doc     string  `json:"doc"`
}

// -----------------------------------------------------------------------------
// Loader: Fetch + map into internal model
// -----------------------------------------------------------------------------

// LoadSchemaFromGroq fetches schema JSON from Groq API and builds a Schema.
func LoadSchemaFromGroq(ctx context.Context, endpoint string, apiKey string) (Schema, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
    if err != nil {
        return nil, err
    }
    if apiKey != "" {
        req.Header.Set("Authorization", "Bearer "+apiKey)
    }
    req.Header.Set("Accept", "application/json")

    client := &http.Client{Timeout: 15 * time.Second}
    res, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("Groq API request failed: %w", err)
    }
    defer res.Body.Close()

    if res.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
        return nil, fmt.Errorf("Groq API returned %d: %s", res.StatusCode, string(body))
    }

    var payload groqSchemaResponse
    if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
        return nil, fmt.Errorf("failed to decode Groq schema JSON: %w", err)
    }

    // Map into our internal structs
    s := &schemaImpl{
        version:  payload.Version,
        entities: make([]entityImpl, len(payload.Entities)),
    }

    for i, ge := range payload.Entities {
        ent := entityImpl{
            name:      ge.Name,
            table:     ge.Table,
            doc:       ge.Doc,
            fields:    make([]fieldImpl, len(ge.Fields)),
            relations: make([]relationImpl, len(ge.Relations)),
        }

        for fi, gf := range ge.Fields {
            ent.fields[fi] = fieldImpl{
                name:         gf.Name,
                column:       gf.Column,
                kind:         gf.Kind,
                format:       gf.Format,
                nullable:     gf.Nullable,
                primaryKey:   gf.PrimaryKey,
                unique:       gf.Unique,
                defaultValue: gf.Default,
                maxLen:       gf.MaxLen,
                scale:        gf.Scale,
                precision:    gf.Precision,
                tags:         gf.Tags,
                doc:          gf.Doc,
            }
        }

        for ri, gr := range ge.Relations {
            ent.relations[ri] = relationImpl{
                rtype:   gr.Type,
                fromEnt: gr.FromEnt,
                fromFld: gr.FromFld,
                toEnt:   gr.ToEnt,
                toFld:   gr.ToFld,
                through: gr.Through,
                doc:     gr.Doc,
            }
        }

        s.entities[i] = ent
    }

    return s, nil
}
