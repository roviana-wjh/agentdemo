"""Caesar 与 Playfair 密码算法实验工具。"""
from __future__ import annotations
import argparse, collections, string, sys, tkinter as tk
from pathlib import Path
from tkinter import filedialog, messagebox, ttk

ROOT = Path(__file__).resolve().parent
DEFAULT_TEXT = ROOT / "示例明文.txt"

def caesar_transform(text: str, shift: int) -> str:
    """英文字母循环位移，其他字符保持不变。"""
    out=[]
    for ch in text:
        if 'A'<=ch<='Z': out.append(chr((ord(ch)-65+shift)%26+65))
        elif 'a'<=ch<='z': out.append(chr((ord(ch)-97+shift)%26+97))
        else: out.append(ch)
    return ''.join(out)

def frequency(text: str) -> str:
    """统计字符频率并显示中文空白标签。"""
    c=collections.Counter(ch.upper() if ch.isascii() and ch.isalpha() else ch for ch in text)
    labels={' ':'[空格]','\n':'[换行]','\t':'[制表]'}
    return '\n'.join(f'{labels.get(k,k)}: {v}' for k,v in sorted(c.items(), key=lambda x:(-x[1],x[0]))) or '（没有可统计字符）'

def normalize_playfair(text: str) -> str:
    """只保留字母，并将 J 合并为 I。"""
    return ''.join('I' if ch.upper()=='J' else ch.upper() for ch in text if ch.isascii() and ch.isalpha())

def playfair_square(key: str):
    """构造 5x5 密钥方阵。"""
    letters=[]
    for ch in normalize_playfair(key)+string.ascii_uppercase:
        ch='I' if ch=='J' else ch
        if ch not in letters: letters.append(ch)
    return [letters[i:i+5] for i in range(0,25,5)]

def playfair_prepare(text: str) -> str:
    """拆分双字母组，重复字母间插入 X，奇数长度末尾补 X。"""
    s=normalize_playfair(text); out=[]; i=0
    while i<len(s):
        a=s[i]
        if i+1>=len(s): out.append(a+'X'); i+=1
        elif s[i+1]==a: out.append(a+'X'); i+=1
        else: out.append(a+s[i+1]); i+=2
    return ''.join(out)

def playfair_encrypt(text: str, key: str) -> str:
    """同行右移、同列下移、矩形换位。"""
    sq=playfair_square(key); pos={sq[r][c]:(r,c) for r in range(5) for c in range(5)}; p=playfair_prepare(text); out=[]
    for i in range(0,len(p),2):
        a,b=p[i:i+2]; ra,ca=pos[a]; rb,cb=pos[b]
        if ra==rb: out += [sq[ra][(ca+1)%5],sq[rb][(cb+1)%5]]
        elif ca==cb: out += [sq[(ra+1)%5][ca],sq[(rb+1)%5][cb]]
        else: out += [sq[ra][cb],sq[rb][ca]]
    return ''.join(out)

class CryptoApp:
    """Tkinter 图形界面。"""
    def __init__(self, root):
        self.root=root; root.title('密码算法实验室 - Caesar / Playfair'); root.geometry('1050x700')
        self.file_var=tk.StringVar(value=str(DEFAULT_TEXT)); self.shift_var=tk.IntVar(value=3); self.key_var=tk.StringVar(value='MONARCHY'); self.status=tk.StringVar(value='请选择文件或直接编辑明文'); self.build()
    def build(self):
        top=ttk.Frame(self.root,padding=8); top.pack(fill='x'); ttk.Label(top,text='数据文件').pack(side='left'); ttk.Entry(top,textvariable=self.file_var,width=55).pack(side='left',padx=5); ttk.Button(top,text='浏览',command=self.browse).pack(side='left'); ttk.Button(top,text='读取',command=self.load).pack(side='left',padx=4); ttk.Button(top,text='保存密文',command=self.save).pack(side='left')
        bar=ttk.LabelFrame(self.root,text='参数与操作',padding=8); bar.pack(fill='x',padx=8); ttk.Label(bar,text='Caesar 位移').pack(side='left'); ttk.Spinbox(bar,from_=-25,to=25,textvariable=self.shift_var,width=6).pack(side='left',padx=4); ttk.Button(bar,text='Caesar 加密',command=self.caesar_encrypt).pack(side='left'); ttk.Button(bar,text='Caesar 解密',command=self.caesar_decrypt).pack(side='left',padx=3); ttk.Button(bar,text='穷举攻击',command=self.bruteforce).pack(side='left'); ttk.Label(bar,text='Playfair 密钥').pack(side='left',padx=(18,3)); ttk.Entry(bar,textvariable=self.key_var,width=14).pack(side='left'); ttk.Button(bar,text='Playfair 加密',command=self.playfair_encrypt_action).pack(side='left',padx=4)
        panes=ttk.PanedWindow(self.root,orient='horizontal'); panes.pack(fill='both',expand=True,padx=8,pady=8); left=ttk.LabelFrame(panes,text='明文 / 输入',padding=4); right=ttk.LabelFrame(panes,text='密文 / 输出',padding=4); panes.add(left,weight=1); panes.add(right,weight=1); self.plain=tk.Text(left,wrap='word'); self.plain.pack(fill='both',expand=True); self.plain.insert('1.0','THE QUICK BROWN FOX JUMPS OVER THE LAZY DOG.'); self.cipher=tk.Text(right,wrap='word'); self.cipher.pack(fill='both',expand=True); self.stats=tk.Text(self.root,height=9,wrap='word'); self.stats.pack(fill='x',padx=8); ttk.Label(self.root,textvariable=self.status,relief='sunken',anchor='w').pack(fill='x')
    def browse(self):
        p=filedialog.askopenfilename(filetypes=[('文本文件','*.txt'),('所有文件','*.*')]); self.file_var.set(p) if p else None
    def load(self):
        try: self.plain.delete('1.0','end'); self.plain.insert('1.0',Path(self.file_var.get()).read_text(encoding='utf-8')); self.status.set('已读取数据文件')
        except Exception as e: messagebox.showerror('读取失败',str(e))
    def current(self): return self.plain.get('1.0','end-1c')
    def show(self,name,result):
        self.cipher.delete('1.0','end'); self.cipher.insert('1.0',result); self.stats.delete('1.0','end'); self.stats.insert('1.0',f'[{name}]\n\n明文频率：\n{frequency(self.current())}\n\n密文频率：\n{frequency(result)}'); self.status.set(name+'完成')
    def caesar_encrypt(self): self.show('Caesar 加密',caesar_transform(self.current(),self.shift_var.get()))
    def caesar_decrypt(self): self.show('Caesar 解密',caesar_transform(self.current(),-self.shift_var.get()))
    def playfair_encrypt_action(self): self.show('Playfair 加密',playfair_encrypt(self.current(),self.key_var.get()))
    def bruteforce(self):
        cipher=self.cipher.get('1.0','end-1c') or self.current(); self.stats.delete('1.0','end'); self.stats.insert('1.0','[Caesar 穷举攻击]\n'+'\n'.join(f'位移 {k:2d}: {caesar_transform(cipher,-k)}' for k in range(26))); self.status.set('穷举完成，请结合语义判断')
    def save(self):
        p=filedialog.asksaveasfilename(defaultextension='.txt',filetypes=[('文本文件','*.txt')]); Path(p).write_text(self.cipher.get('1.0','end-1c'),encoding='utf-8') if p else None

def cli(a):
    text=Path(a.input).read_text(encoding='utf-8'); result=caesar_transform(text,a.shift if a.action=='encrypt' else -a.shift) if a.algorithm=='caesar' else playfair_encrypt(text,a.key); print(result); print('\n--- 原文频率 ---\n'+frequency(text)+'\n--- 结果频率 ---\n'+frequency(result))
    if a.bruteforce:
        print('\n--- 穷举结果 ---'); [print(f'位移 {k:2d}: {caesar_transform(result,-k)}') for k in range(26)]
    if a.output: Path(a.output).write_text(result,encoding='utf-8')

def main():
    p=argparse.ArgumentParser(description='Caesar 与 Playfair 密码实验工具'); p.add_argument('--cli',action='store_true'); p.add_argument('--input',default=str(DEFAULT_TEXT)); p.add_argument('--output'); p.add_argument('--algorithm',choices=['caesar','playfair'],default='caesar'); p.add_argument('--action',choices=['encrypt','decrypt'],default='encrypt'); p.add_argument('--shift',type=int,default=3); p.add_argument('--key',default='MONARCHY'); p.add_argument('--bruteforce',action='store_true'); a=p.parse_args()
    if a.cli: cli(a); return
    try: root=tk.Tk(); CryptoApp(root); root.mainloop()
    except tk.TclError as e: print(f'无法启动图形界面：{e}，请改用 --cli',file=sys.stderr); sys.exit(1)

if __name__=='__main__': main()
