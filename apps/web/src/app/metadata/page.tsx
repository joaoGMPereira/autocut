'use client';

import { Anton } from 'next/font/google';
import { useState } from 'react';
import { Button } from '@/components/ui/button';

const anton = Anton({ weight: '400', subsets: ['latin'], display: 'swap' });

const DRAFT_KEY = 'autocut-metadata-draft';

const YOUTUBE_CATEGORIES = [
  'Entertainment',
  'Education',
  'Science & Technology',
  'Gaming',
  'Music',
  'Sports',
  'News & Politics',
  'Howto & Style',
  'Travel & Events',
] as const;

export default function MetadataPage() {
  const [metaTitle, setMetaTitle] = useState('');
  const [description, setDescription] = useState('');
  const [tags, setTags] = useState<string[]>([]);
  const [category, setCategory] = useState('');
  const [tagInput, setTagInput] = useState('');
  const [savedMessage, setSavedMessage] = useState(false);

  const addTag = (value: string) => {
    const trimmed = value.trim();
    if (trimmed && !tags.includes(trimmed)) {
      setTags((prev) => [...prev, trimmed]);
    }
    setTagInput('');
  };

  const removeTag = (tag: string) => {
    setTags((prev) => prev.filter((t) => t !== tag));
  };

  const handleTagKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      addTag(tagInput);
    }
  };

  const saveDraft = () => {
    localStorage.setItem(
      DRAFT_KEY,
      JSON.stringify({ title: metaTitle, description, tags, category }),
    );
    setSavedMessage(true);
    setTimeout(() => setSavedMessage(false), 2000);
  };

  const loadDraft = () => {
    const raw = localStorage.getItem(DRAFT_KEY);
    if (!raw) return;
    try {
      const draft = JSON.parse(raw) as {
        title: string;
        description: string;
        tags: string[];
        category: string;
      };
      setMetaTitle(draft.title ?? '');
      setDescription(draft.description ?? '');
      setTags(draft.tags ?? []);
      setCategory(draft.category ?? '');
    } catch {
      /* ignore corrupt drafts */
    }
  };

  return (
    <div className="px-10 py-8 flex flex-col gap-7">
      {/* Header */}
      <h1 className={`${anton.className} text-[28px] tracking-[1.5px] text-[#F0F0F8]`}>
        METADATA
      </h1>

      {/* Form card */}
      <div className="rounded-xl bg-card border border-border p-5 flex flex-col gap-5">
        <h2 className="text-sm font-semibold">Video Metadata</h2>

        {/* Title */}
        <div>
          <label className="text-xs text-[#5C5C80] mb-1 block">Title</label>
          <input
            type="text"
            value={metaTitle}
            onChange={(e) => setMetaTitle(e.target.value)}
            placeholder="Video title"
            className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
          />
        </div>

        {/* Description */}
        <div>
          <label className="text-xs text-[#5C5C80] mb-1 block">Description</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Video description"
            className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm min-h-[100px] resize-none focus:outline-none focus:border-[#00D4FF]/60"
          />
        </div>

        {/* Tags chip input */}
        <div>
          <label className="text-xs text-[#5C5C80] mb-1 block">Tags</label>
          <div className="flex flex-col gap-2">
            {tags.length > 0 && (
              <div className="flex flex-wrap gap-1.5">
                {tags.map((tag) => (
                  <span
                    key={tag}
                    className="flex items-center gap-1 bg-[#00D4FF]/10 text-[#00D4FF] text-xs px-2 py-0.5 rounded-md"
                  >
                    <span>{tag}</span>
                    <button
                      onClick={() => removeTag(tag)}
                      className="hover:text-white leading-none"
                      aria-label={`Remove tag ${tag}`}
                    >
                      ×
                    </button>
                  </span>
                ))}
              </div>
            )}
            <input
              type="text"
              value={tagInput}
              onChange={(e) => setTagInput(e.target.value)}
              onKeyDown={handleTagKeyDown}
              placeholder="Type a tag and press Enter"
              className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60"
            />
          </div>
        </div>

        {/* Category */}
        <div>
          <label className="text-xs text-[#5C5C80] mb-1 block">Category</label>
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="w-full rounded-lg bg-[#1A1A26] border border-border px-3 py-2 text-sm focus:outline-none focus:border-[#00D4FF]/60 appearance-none"
          >
            <option value="">Select a category…</option>
            {YOUTUBE_CATEGORIES.map((cat) => (
              <option key={cat} value={cat}>
                {cat}
              </option>
            ))}
          </select>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          <Button onClick={saveDraft}>{savedMessage ? 'Saved!' : 'Save Draft'}</Button>
          <Button variant="outline" onClick={loadDraft}>
            Load Draft
          </Button>
        </div>

        {/* Footer */}
        <div className="pt-1">
          <span className="bg-[#1A1A26] text-[#5C5C80] text-xs px-2 py-0.5 rounded-md">
            YouTube API integration coming soon
          </span>
        </div>
      </div>
    </div>
  );
}
