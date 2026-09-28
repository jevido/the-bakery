import { mount } from 'svelte'
import '@bakery/ui/theme.css'
import './console.css'
import App from './App.svelte'

mount(App, { target: document.getElementById('app')! })
