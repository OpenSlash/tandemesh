import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FileText, Plus, Trash2 } from 'lucide-react'
import { api, type Document } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { useI18n } from '../lib/i18n'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'

export default function DocumentsPage() {
  const { workspaceId } = useWorkspace()
  const { t } = useI18n()
  const qc = useQueryClient()
  const { data } = useQuery({
    queryKey: ['documents', workspaceId],
    queryFn: () => api.documents(workspaceId),
  })
  const documents = data?.documents ?? []
  const [selectedId, setSelectedId] = useState('')
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [tags, setTags] = useState('')

  const selected = documents.find((doc) => doc.id === selectedId)

  useEffect(() => {
    if (selected) {
      setTitle(selected.title)
      setBody(selected.body)
      setTags((selected.tags ?? []).join(', '))
    }
  }, [selectedId, selected?.updatedAt])

  const refresh = () => qc.invalidateQueries({ queryKey: ['documents', workspaceId] })
  const save = useMutation({
    mutationFn: async () => {
      const parsedTags = tags.split(',').map((tag) => tag.trim()).filter(Boolean)
      if (selectedId) {
        return api.updateDocument(selectedId, { title, body, tags: parsedTags })
      }
      return api.createDocument({ workspaceId, title, body, tags: parsedTags })
    },
    onSuccess: (doc: Document) => {
      setSelectedId(doc.id)
      refresh()
    },
  })
  const remove = useMutation({
    mutationFn: (id: string) => api.deleteDocument(id),
    onSuccess: () => {
      setSelectedId('')
      setTitle('')
      setBody('')
      setTags('')
      refresh()
    },
  })

  return (
    <div className="flex h-full">
      <div className="flex w-[300px] shrink-0 flex-col border-r border-line">
        <div className="flex items-center justify-between px-4 py-4">
          <h1 className="text-[15px] font-semibold">{t('nav.documents')}</h1>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => {
              setSelectedId('')
              setTitle('')
              setBody('')
              setTags('')
            }}
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>
        <div className="flex-1 overflow-y-auto px-2 pb-4">
          {documents.length === 0 && (
            <p className="px-3 py-8 text-center text-[12px] text-muted">{t('documents.empty')}</p>
          )}
          {documents.map((doc) => (
            <button
              key={doc.id}
              onClick={() => setSelectedId(doc.id)}
              className={`mb-1 flex w-full items-start gap-2 rounded-md px-3 py-2 text-left ${
                doc.id === selectedId ? 'bg-panel2 text-fg' : 'text-muted hover:bg-panel2/50 hover:text-fg'
              }`}
            >
              <FileText className="mt-0.5 h-4 w-4 shrink-0" />
              <div className="min-w-0">
                <div className="truncate text-[12.5px] font-medium">{doc.title}</div>
                <div className="truncate text-[11px] text-muted/60">
                  {new Date(doc.updatedAt).toLocaleDateString()}
                </div>
              </div>
            </button>
          ))}
        </div>
      </div>

      <div className="flex min-w-0 flex-1 flex-col px-6 py-5">
        <div className="flex items-center gap-2">
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder={t('documents.titlePlaceholder')}
            className="max-w-[420px]"
          />
          <Input
            value={tags}
            onChange={(e) => setTags(e.target.value)}
            placeholder={t('documents.tagsPlaceholder')}
            className="max-w-[240px]"
          />
          <Button
            size="sm"
            onClick={() => save.mutate()}
            disabled={save.isPending || title.trim() === ''}
          >
            {t('common.save')}
          </Button>
          {selectedId && (
            <Button size="sm" variant="ghost" onClick={() => remove.mutate(selectedId)}>
              <Trash2 className="h-4 w-4" />
            </Button>
          )}
        </div>
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder={t('documents.bodyPlaceholder')}
          className="mt-3 min-h-[420px] flex-1 resize-none rounded-lg border border-line bg-panel p-4 font-mono text-[13px] leading-6 text-fg outline-none focus:border-accent"
        />
      </div>
    </div>
  )
}
