import { mount } from 'svelte'
import '@bakery/ui/theme.css'
import './site.css'
import App from './App.svelte'

mount(App, { target: document.getElementById('app')! })
