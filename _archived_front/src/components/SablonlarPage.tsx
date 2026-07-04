import React, { useState } from 'react';
import { 
  FileText, 
  CheckCircle2, 
  Eye, 
  Plus, 
  X,
  MessageSquare,
  Globe,
  Tag
} from 'lucide-react';
import { Template } from '../types';

interface SablonlarPageProps {
  templates: Template[];
  setTemplates: React.Dispatch<React.SetStateAction<Template[]>>;
}

export default function SablonlarPage({
  templates,
  setTemplates
}: SablonlarPageProps) {
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);

  // New template form state
  const [newName, setNewName] = useState('');
  const [newCategory, setNewCategory] = useState<'UTILITY' | 'MARKETING'>('UTILITY');
  const [newLanguage, setNewLanguage] = useState('tr');
  const [newBody, setNewBody] = useState('');

  const handleCreateTemplate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newName.trim() || !newBody.trim()) return;

    // Formatting template name: lower_snake_case
    const formattedName = newName
      .trim()
      .toLowerCase()
      .replace(/\s+/g, '_')
      .replace(/[^a-z0-9_]/g, '');

    const newTemp: Template = {
      id: 't-new-' + Date.now(),
      name: formattedName,
      category: newCategory,
      language: newLanguage,
      status: 'APPROVED',
      body: newBody
    };

    setTemplates(prev => [...prev, newTemp]);
    setShowAddModal(false);
    
    // Clear form
    setNewName('');
    setNewBody('');
  };

  // Helper to render mock formatted text (e.g. bolding *word* or filling placeholders)
  const getFilledPreview = (body: string) => {
    // Replace standard variables with mock human-readable values to show active translation
    return body
      .replace('{{1}}', 'Ahmet Bey')
      .replace('{{2}}', '14:00')
      .replace('{{3}}', '15:00')
      .replace('{{1}}', 'Ahmet Bey') // catch multiple if any
      .replace('{{2}}', '25.07.2026');
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-xs font-bold text-zinc-900 tracking-tight">Onaylı WhatsApp Şablonları</h2>
          <p className="text-[11px] text-zinc-500 mt-1">
            Meta onaylı resmî WhatsApp bulut API şablonları. Bu mesajlar otonom nitelik takibinde kullanılır.
          </p>
        </div>

        <button
          onClick={() => setShowAddModal(true)}
          className="inline-flex items-center gap-1 py-1.5 px-3 bg-zinc-900 text-white hover:bg-zinc-800 text-[11px] font-bold rounded shadow-sm transition-colors self-start sm:self-auto"
        >
          <Plus className="w-3.5 h-3.5" /> Yeni Şablon Tanımla
        </button>
      </div>

      {/* Grid List */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
        {templates.map((temp) => (
          <div 
            key={temp.id}
            className="bg-white border border-zinc-200/80 rounded-lg p-4.5 shadow-sm hover:border-zinc-300 transition-all flex flex-col justify-between text-left"
          >
            <div>
              {/* Badge strip */}
              <div className="flex flex-wrap items-center justify-between gap-2 mb-3">
                <div className="flex items-center gap-1.5">
                  <span className={`
                    text-[9px] font-bold px-2 py-0.5 rounded-md border
                    ${temp.category === 'UTILITY' 
                      ? 'bg-zinc-100 border-zinc-200 text-zinc-700' 
                      : 'bg-amber-50 border-amber-200 text-amber-700'
                    }
                  `}>
                    {temp.category}
                  </span>
                  <span className="text-[9px] font-mono font-bold bg-zinc-50 border border-zinc-200 text-zinc-500 px-1.5 py-0.2 rounded">
                    {temp.language.toUpperCase()}
                  </span>
                </div>

                <span className="inline-flex items-center gap-1 text-[10px] font-bold text-emerald-700 bg-emerald-50 border border-emerald-200/50 px-2 py-0.5 rounded-full">
                  <CheckCircle2 className="w-3 h-3 text-emerald-500" /> Onaylı (Meta)
                </span>
              </div>

              {/* Template title */}
              <h3 className="text-xs font-bold text-zinc-900 font-mono tracking-tight mb-2.5 break-all">
                {temp.name}
              </h3>

              {/* Template Body display */}
              <div className="p-3 bg-zinc-50 border border-zinc-100 rounded text-[11px] text-zinc-600 leading-relaxed font-sans mb-4 min-h-[72px]">
                {temp.body}
              </div>
            </div>

            {/* View Preview button */}
            <div className="border-t border-zinc-100 pt-3 flex items-center justify-between">
              <span className="text-[10px] text-zinc-400 font-medium">
                Değişken sayısı: { (temp.body.match(/\{\{\d\}\}/g) || []).length }
              </span>
              <button
                onClick={() => setSelectedTemplate(temp)}
                className="inline-flex items-center gap-1 text-[10.5px] font-bold text-zinc-800 hover:text-zinc-900 hover:underline"
              >
                <Eye className="w-3.5 h-3.5" /> WhatsApp Önizleme
              </button>
            </div>
          </div>
        ))}
      </div>

      {/* 1. Preview Modal */}
      {selectedTemplate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/20 p-4">
          <div className="bg-zinc-100 border border-zinc-300 rounded-lg shadow-xl w-full max-w-sm overflow-hidden animate-in fade-in zoom-in-95 duration-100">
            {/* WhatsApp Phone-like header mock */}
            <div className="bg-[#075e54] text-white px-4 py-3 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="w-7 h-7 bg-white/20 rounded-full flex items-center justify-center">
                  <MessageSquare className="w-4 h-4 text-white" />
                </div>
                <div>
                  <div className="text-xs font-bold">BrainMeta WhatsApp API</div>
                  <div className="text-[9px] text-emerald-100">resmi işletme hesabı</div>
                </div>
              </div>
              <button 
                onClick={() => setSelectedTemplate(null)}
                className="p-1 text-white/80 hover:text-white rounded"
              >
                <X className="w-4.5 h-4.5" />
              </button>
            </div>

            {/* Chat Body Mock */}
            <div className="p-4 bg-[#e5ddd5] min-h-[220px] flex flex-col justify-end text-left">
              <div className="bg-white p-3 rounded-lg shadow-sm max-w-[85%] relative self-start text-xs text-zinc-800 leading-relaxed rounded-tl-none">
                {/* Visual bubble point */}
                <div className="absolute top-0 -left-1.5 w-2 h-2.5 bg-white clip-path-whatsapp" />
                
                {/* Content with simulated parameters */}
                <p className="whitespace-pre-wrap">{getFilledPreview(selectedTemplate.body)}</p>
                
                <span className="block text-[9px] text-zinc-400 text-right mt-1.5 font-mono">
                  {new Date().toLocaleTimeString('tr-TR', { hour: '2-digit', minute: '2-digit' })}
                </span>
              </div>
            </div>

            {/* Actions */}
            <div className="bg-white p-3 text-center border-t border-zinc-200">
              <p className="text-[10px] text-zinc-400 mb-2">
                * Bu görünüm, Meta değişkenleri doldurularak simüle edilmiştir.
              </p>
              <button
                onClick={() => setSelectedTemplate(null)}
                className="w-full py-1.5 bg-zinc-900 text-white hover:bg-zinc-800 text-xs font-bold rounded"
              >
                Kapat
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 2. Add New Template Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4">
          <div className="bg-white border border-zinc-200 rounded-lg shadow-xl w-full max-w-md overflow-hidden animate-in fade-in zoom-in-95 duration-150">
            {/* Header */}
            <div className="px-4 py-3 border-b border-zinc-100 flex items-center justify-between bg-zinc-50">
              <span className="text-xs font-bold text-zinc-900">
                Yeni Meta WhatsApp Şablon Tanımı
              </span>
              <button 
                onClick={() => setShowAddModal(false)}
                className="p-1 text-zinc-400 hover:text-zinc-600 rounded"
              >
                <X className="w-4.5 h-4.5" />
              </button>
            </div>

            {/* Form */}
            <form onSubmit={handleCreateTemplate} className="p-5 space-y-4 text-left">
              <div>
                <label className="block text-[11px] font-bold text-zinc-700 mb-1">
                  ŞABLON ADI (KÜÇÜK HARF, ALT TİRE)
                </label>
                <input
                  type="text"
                  required
                  placeholder="kampanya_implant_kasim"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  className="w-full bg-zinc-50 border border-zinc-200 rounded px-3 py-2 text-xs font-mono focus:outline-none focus:border-zinc-400 focus:bg-white"
                />
              </div>

              <div className="grid grid-cols-2 gap-3.5">
                <div>
                  <label className="block text-[11px] font-bold text-zinc-700 mb-1">
                    KATEGORİ
                  </label>
                  <select
                    value={newCategory}
                    onChange={(e) => setNewCategory(e.target.value as 'UTILITY' | 'MARKETING')}
                    className="w-full bg-zinc-50 border border-zinc-200 rounded px-2.5 py-1.5 text-xs font-medium focus:outline-none"
                  >
                    <option value="UTILITY">UTILITY (Bilgilendirme)</option>
                    <option value="MARKETING">MARKETING (Tanıtım)</option>
                  </select>
                </div>

                <div>
                  <label className="block text-[11px] font-bold text-zinc-700 mb-1">
                    DİL (KODU)
                  </label>
                  <input
                    type="text"
                    required
                    value={newLanguage}
                    onChange={(e) => setNewLanguage(e.target.value)}
                    className="w-full bg-zinc-50 border border-zinc-200 rounded px-3 py-1.5 text-xs font-semibold focus:outline-none"
                  />
                </div>
              </div>

              <div>
                <label className="block text-[11px] font-bold text-zinc-700 mb-1 flex items-center justify-between">
                  <span>ŞABLON GÖVDESİ</span>
                  <span className="text-[10px] text-zinc-400 font-normal">Değişkenler için {"{{1}}"}, {"{{2}}"} kullanın.</span>
                </label>
                <textarea
                  required
                  rows={4}
                  placeholder="Merhaba {{1}}, talebiniz üzere en uygun implant gününü planlamak için bugun saat {{2}}'de arayacağız."
                  value={newBody}
                  onChange={(e) => setNewBody(e.target.value)}
                  className="w-full bg-zinc-50 border border-zinc-200 rounded px-3 py-2 text-xs font-semibold focus:outline-none focus:border-zinc-400 focus:bg-white"
                />
              </div>

              {/* Action Buttons */}
              <div className="pt-3 border-t border-zinc-100 flex items-center justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setShowAddModal(false)}
                  className="px-3.5 py-1.5 text-xs font-semibold text-zinc-500 hover:text-zinc-700"
                >
                  Vazgeç
                </button>
                <button
                  type="submit"
                  className="px-4 py-1.5 bg-zinc-900 hover:bg-zinc-800 text-white text-xs font-bold rounded shadow-sm"
                >
                  Meta Onayına Gönder
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

    </div>
  );
}
