SecBan Alert from {{ $.Host }}

{{ $.Offender.Timestamp.Format "2006-01-02 15:04:05" }} banned {{ $.IP }} according to rule {{ $.Rule }}
{{ range $.Offender.Entries }}
    - {{ .Timestamp.Format "2006-01-02 15:04:05" }}:  {{ .Message }}
{{ end }}